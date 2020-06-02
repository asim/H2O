package dao

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/gossie/src/gossie"
)

/* This is how we store stuff:
   RowKey: 9Bf3155ejPvwuzhU33YUmYOAz7tVI1dv+h+vBCpWivvV68Sz8PVMH3OaeM/+h3RM
   => (column=session,
   value={
   		"key":"9Bf3155ejPvwuzhU33YUmYOAz7tVI1dv+h+vBCpWivvV68Sz8PVMH3OaeM\/+h3RM",
   		"createdTimestamp":1378377733,
   		"token":{
   			"createdTimestamp":1378377733,
   			"authMechanism":"admin",
   			"deviceType":"cli",
   			"id":"dave",
   			"expiryTimestamp":1378406533,
   			"autoRenewTimestamp":null,
   			"rolePatterns": ["ADMIN"],
   			"roleCollection":{
   				"ADMIN":"ADMIN"
   			},
   			"signature":"…"
   		}
   	},
   	timestamp=1378377733437966) */

type storedSession struct {
	Id          string `cf:"sessions" key:"Id"`
	SessionData []byte `name:"session"`
}

type encodedSession struct {
	Key              string       `json:"key"`
	CreatedTimestamp int64        `json:"createdTimestamp"`
	Token            encodedToken `json:"token"`
}

type encodedToken struct {
	CreatedTimestamp   int64  `json:"createdTimestamp"`
	AuthMechanism      string `json:"authMechanism"`
	DeviceType         string `json:"deviceType"`
	Id                 string `json:"id"`
	ExpiryTimestamp    int64  `json:"expiryTimestamp"`
	AutoRenewTimestamp *int64 `json:"autoRenewTimestamp"`
	Signature          string `json:"signature"`
	// Ordered set of role patterns granted to the user
	RolePatterns []string `json:"rolePatterns"`
	// Deprecated: has been replaced with RolePatterns (which is ordered), but maintained for backwards compatibility
	RoleCollection map[string]string `json:"roleCollection"`
}

type storedUser struct {
	Id                    string    `cf:"users" key:"Id"`
	App                   string    `name:"app"`
	Uid                   string    `name:"uid"`
	Ids                   []byte    `name:"ids"`
	Created               time.Time `name:"createdTimestamp"`
	Roles                 []byte    `name:"roles"`
	PasswordHistory       []byte    `name:"passwordHistory"`
	Password              []byte    `name:"password"`
	PasswordChange        time.Time `name:"passwordChangeTimestamp"`
	AccountExpirationDate string    `name:"accountExpirationDate"`
	Status                string    `name:"status"`
}

// needed due to the fact PHP encodes empty object as [] rather than {} so Go complains
// HTWO-319

type encodedSessionNoRoles struct {
	Key              string              `json:"key"`
	CreatedTimestamp int64               `json:"createdTimestamp"`
	Token            encodedTokenNoRoles `json:"token"`
}
type encodedTokenNoRoles struct {
	CreatedTimestamp   int64    `json:"createdTimestamp"`
	AuthMechanism      string   `json:"authMechanism"`
	DeviceType         string   `json:"deviceType"`
	Id                 string   `json:"id"`
	ExpiryTimestamp    int64    `json:"expiryTimestamp"`
	AutoRenewTimestamp *int64   `json:"autoRenewTimestamp"`
	RolePatterns       []string `json:"rolePatterns"`
	RoleCollection     []string `json:"roleCollection"`
	Signature          string   `json:"signature"`
}

// METHODS

// marshalSession turns a session domain object into a row for storage
func marshalSession(sess *domain.Session) (*gossie.Row, error) {
	// Create a map of roles for RolesCollection backwards compatibility
	rolesMap := make(map[string]string)
	for _, r := range sess.Token.Roles {
		rolesMap[r] = r
	}

	encSess := &encodedSession{
		Key:              sess.Id,
		CreatedTimestamp: sess.Created.Unix(),
		Token: encodedToken{
			CreatedTimestamp:   sess.Token.Created.Unix(),
			AuthMechanism:      sess.Token.AuthMechanism,
			DeviceType:         sess.Token.DeviceType,
			Id:                 sess.Token.Id,
			ExpiryTimestamp:    sess.Token.Expires.Unix(),
			AutoRenewTimestamp: optTimeToUnix(sess.Token.AutoRenew),
			RolePatterns:       sess.Token.Roles,
			RoleCollection:     rolesMap,
			Signature:          sess.Token.Signature,
		},
	}
	jsonBytes, err := json.Marshal(encSess)
	if err != nil {
		return nil, err
	}

	stored := &storedSession{
		Id:          sess.Id,
		SessionData: jsonBytes,
	}
	row, err := sessionMapping.Map(stored)
	if err != nil {
		return nil, err
	}

	return row, nil
}

// optTimeToUnix will take a time and return *int64 unless it's zero time in which case nil
func optTimeToUnix(t time.Time) *int64 {
	if t.IsZero() {
		return nil
	}
	i := t.Unix()
	return &i
}

// unmarshalSession yields a session domain object from a row
func unmarshalSession(row *gossie.Row) (*domain.Session, error) {
	stored := &storedSession{}
	err := sessionMapping.Unmap(stored, &cassandra.SingleRowProvider{Row: row})
	if err != nil && err != gossie.Done {
		return nil, fmt.Errorf("Error unmapping row: %s", err.Error())
	}

	sess, err := unmarshalDefaultSessionData(stored.SessionData)
	if err == nil {
		return sess, nil
	}

	// try backup plan
	if sess, backuperr := unmarshalBorkedSessionData(stored.SessionData); backuperr == nil {
		return sess, nil
	}

	// return original err
	return nil, err
}

// unmarshalDefaultSessionData is the normal path, for how things get encoded when they have roles (nearly always)
func unmarshalDefaultSessionData(data []byte) (*domain.Session, error) {
	encoded := &encodedSession{}
	if err := json.Unmarshal(data, encoded); err != nil {
		return nil, err
	}

	// Deal with legacy tokens which will have data in RoleCollection but not RolePatterns
	var roles []string
	if len(encoded.Token.RolePatterns) == 0 && len(encoded.Token.RoleCollection) > 0 {
		roles = make([]string, len(encoded.Token.RoleCollection))
		i := 0
		for k := range encoded.Token.RoleCollection {
			roles[i] = k
			i++
		}
		// Ensure the roles are sorted (lexicographically, in this case)
		sort.Strings(roles)
	} else {
		roles = encoded.Token.RolePatterns
	}

	return &domain.Session{
		Id:      encoded.Key,
		Created: unixToTime(encoded.CreatedTimestamp),
		Token: domain.Token{
			Created:       unixToTime(encoded.Token.CreatedTimestamp),
			AuthMechanism: encoded.Token.AuthMechanism,
			DeviceType:    encoded.Token.DeviceType,
			Id:            encoded.Token.Id,
			Expires:       unixToTime(encoded.Token.ExpiryTimestamp),
			AutoRenew:     optUnixToTime(encoded.Token.AutoRenewTimestamp),
			Roles:         roles,
			Signature:     encoded.Token.Signature,
		},
	}, nil
}

// unmarshalBorkedSessionData is the backup plan for when we have a borked token with no roles caused by PHP encoding
// empty map as []
func unmarshalBorkedSessionData(data []byte) (*domain.Session, error) {
	encoded := &encodedSessionNoRoles{}
	if err := json.Unmarshal(data, encoded); err != nil {
		return nil, err
	}

	// MUST have empty string array -- since we only expect this when PHP has messed up roles serde
	if len(encoded.Token.RoleCollection) > 0 {
		return nil, fmt.Errorf("Backup session serde failed because not expecting any roles in RoleCollection; got %d",
			len(encoded.Token.RoleCollection))
	}

	return &domain.Session{
		Id:      encoded.Key,
		Created: unixToTime(encoded.CreatedTimestamp),
		Token: domain.Token{
			Created:       unixToTime(encoded.Token.CreatedTimestamp),
			AuthMechanism: encoded.Token.AuthMechanism,
			DeviceType:    encoded.Token.DeviceType,
			Id:            encoded.Token.Id,
			Expires:       unixToTime(encoded.Token.ExpiryTimestamp),
			AutoRenew:     optUnixToTime(encoded.Token.AutoRenewTimestamp),
			Roles:         encoded.Token.RolePatterns,
			Signature:     encoded.Token.Signature,
		},
	}, nil
}

// writeSession turns a session into mutations
func writeSession(sess *domain.Session, writer gossie.Writer) error {
	row, err := marshalSession(sess)
	if err != nil {
		return fmt.Errorf("Write error marshaling session: %v", err)
	}

	// write this row for all row keys required
	rowKeys := sessionToRowKeys(sess)
	for _, rowKey := range rowKeys {
		row.Key = rowKey
		writer.Insert(cfSessions, row)
	}

	return nil
}

// deleteSession turns a session into deletion mutations
func deleteSession(sess *domain.Session, writer gossie.Writer) {
	// delete all rows
	rowKeys := sessionToRowKeys(sess)
	for _, rowKey := range rowKeys {
		writer.Delete(cfSessions, rowKey)
	}
}

// unixToTime turns a UNIX timestamp in seconds into a time.Time where 0 -> zero time
func unixToTime(t int64) time.Time {
	if t == 0 {
		return time.Time{}
	}
	return time.Unix(t, 0)
}

// optUnixToTime turns a UNIX timestamp in seconds into a time.Time where nil or 0 -> zero time
func optUnixToTime(t *int64) time.Time {
	if t == nil || *t == 0 {
		return time.Time{}
	}
	return time.Unix(*t, 0)
}

// marshalUser turns a user domain object into a stored user for storage
func marshalUser(user *domain.User) *storedUser {
	ids, _ := json.Marshal(user.Ids)
	roles, _ := json.Marshal(user.Roles)
	pwordHist := make([]string, 0)
	for _, hash := range user.PasswordHistory {
		pwordHist = append(pwordHist, base64.StdEncoding.EncodeToString(hash))
	}
	pbytes, _ := json.Marshal(pwordHist)
	return &storedUser{
		Id:                    string(userIdToRowKey(user.App, user.Uid)),
		App:                   string(user.App),
		Uid:                   user.Uid,
		Ids:                   ids,
		Created:               user.Created,
		Roles:                 roles,
		PasswordHistory:       pbytes,
		Password:              user.Password,
		PasswordChange:        user.PasswordChange,
		AccountExpirationDate: user.AccountExpirationDate,
		Status:                user.Status,
	}
}

// unmarshalUser turns a row into a user
func unmarshalUser(row *gossie.Row) (*domain.User, error) {
	stored := &storedUser{}
	err := userMapping.Unmap(stored, &cassandra.SingleRowProvider{Row: row})
	if err != nil && err != gossie.Done {
		return nil, fmt.Errorf("Error unmapping row: %v", err)
	}

	ids := make([]domain.Id, 0)
	json.Unmarshal(stored.Ids, &ids)
	roles := make([]string, 0)
	json.Unmarshal(stored.Roles, &roles)
	pwordHist := make([]string, 0)
	json.Unmarshal(stored.PasswordHistory, &pwordHist)
	pwords := make([][]byte, 0)
	for _, s := range pwordHist {
		b, _ := base64.StdEncoding.DecodeString(s)
		pwords = append(pwords, b)
	}

	return &domain.User{
		App:                   domain.Application(stored.App),
		Uid:                   stored.Uid,
		Ids:                   ids,
		Created:               stored.Created,
		Roles:                 roles,
		PasswordHistory:       pwords,
		Password:              stored.Password,
		PasswordChange:        stored.PasswordChange,
		AccountExpirationDate: stored.AccountExpirationDate,
		Status:                stored.Status,
	}, nil
}

// writeUser maps a user to a mutation, including updating all indexes for additional IDs
func writeUser(user *domain.User, writer gossie.Writer, existingUser *domain.User) error {
	suser := marshalUser(user)
	row, err := userMapping.Map(suser)
	if err != nil {
		return fmt.Errorf("Error mapping user: %v", err)
	}

	writer.Insert(cfUsers, row)

	// store new ids here so we know not to remove them
	newIds := make(map[domain.Id]bool)

	// update for each secondary index
	for _, id := range user.Ids {
		row.Key = userIdToRowKey(user.App, string(id))
		writer.Insert(cfUsers, row)

		// store so we know not to remove it
		newIds[id] = true
	}

	// remove from secondary indexes
	if existingUser != nil {
		for _, id := range existingUser.Ids {
			// skip if still needed
			if newIds[id] {
				continue
			}

			writer.Delete(cfUsers, userIdToRowKey(user.App, string(id)))
		}
	}

	// update timeseries index
	userTs.Map(writer, user, existingUser)

	return nil
}

// deleteUser maps a user to a mutation of deletions, including removing all indexes for additional IDs
func deleteUser(user *domain.User, writer gossie.Writer, uid string, ids []domain.Id) {
	writer.Delete(cfUsers, userIdToRowKey(user.App, uid))

	// deletion for each secondary index
	for _, id := range ids {
		writer.Delete(cfUsers, userIdToRowKey(user.App, string(id)))
	}

	// remove from TS index by removing time element
	userTs.Map(writer, &domain.User{
		Uid: uid,
		App: user.App,
	}, user)
}

func userIdToRowKey(app domain.Application, id string) []byte {
	return []byte(string(app) + separator + id)
}

func userIdsToRowKeys(app domain.Application, ids []string) [][]byte {
	ret := make([][]byte, len(ids))
	for i, id := range ids {
		ret[i] = []byte(string(app) + separator + id)
	}
	return ret
}

func sessionToRowKeys(sess *domain.Session) [][]byte {
	ids := make([][]byte, 2)
	// primary
	ids[0] = []byte(sess.Id)
	// secondary by auth mechanism + device + user id combo (cos we only want to allow one of these to be logged in)
	ids[1] = authMechDeviceUserIdToRowKey(sess.Token.AuthMechanism, sess.Token.DeviceType, sess.Token.Id)

	return ids
}

func authMechDeviceUserIdToRowKey(authMech, deviceType, userId string) []byte {
	return []byte(authMech + separator + deviceType + separator + userId)
}

// unmarshalEndpointAuth turns a gossie row into a slice of EndpointAuth pointers
func unmarshalEndpointAuth(row *gossie.Row) []*domain.EndpointAuth {
	// ROW KEY                                     COL														  VALUE
	// [target service (thing granting access to)] [endpoint (granted to) {separator} (service granted from)] ROLE

	ret := make([]*domain.EndpointAuth, 0)
	for _, column := range row.Columns {
		colNameParts := strings.SplitN(string(column.Name), separator, 2)
		if len(colNameParts) != 2 {
			log.Warnf("Bad column name when unmarshaling EndpointAuth: %v (%#v)", string(column.Name), colNameParts)
			continue
		}

		epa := &domain.EndpointAuth{
			ServiceName:    string(row.Key),
			EndpointName:   colNameParts[0],
			AllowedService: colNameParts[1],
			Role:           string(column.Value),
		}
		ret = append(ret, epa)
	}
	return ret
}

// marshalEndpointAuth marshals an endpoint auth into a writer
func marshalEndpointAuth(epa *domain.EndpointAuth, writer gossie.Writer) {
	// ROW KEY                                     COL														  VALUE
	// [target service (thing granting access to)] [endpoint (granted to) {separator} (service granted from)] ROLE

	row := &gossie.Row{
		Key: []byte(epa.ServiceName),
		Columns: []*gossie.Column{
			{
				Name:  []byte(epa.EndpointName + separator + epa.AllowedService),
				Value: []byte(epa.Role),
			},
		},
	}
	writer.Insert(cfEndpointAuths, row)
}

// deleteEndpointAuth turns an endpoint auth into a deletion mutation and adds to writer
func deleteEndpointAuth(epa *domain.EndpointAuth, writer gossie.Writer) {
	// ROW KEY                                     COL														  VALUE
	// [target service (thing granting access to)] [endpoint (granted to) {separator} (service granted from)] ROLE

	writer.DeleteColumns(cfEndpointAuths, []byte(epa.ServiceName), [][]byte{[]byte(epa.EndpointName + separator + epa.AllowedService)})
}
