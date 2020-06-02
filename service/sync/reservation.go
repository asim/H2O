package sync

import (
	"errors"
	"fmt"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/service/zookeeper"
	gozk "github.com/HailoOSS/go-zookeeper/zk"
)

var (
	ErrReserved = errors.New("[Reservation] Item already reserved")
)

// Reservation will reserve an item given an id.  The reservation must expire and may be released
// at any time by any actor
type Reservation interface {
	Reserve(time.Duration) error
	Release() error
}

type DefaultReservation struct {
	Ttl  time.Duration
	path string
	id   string
	acl  []gozk.ACL
}

type ReservationData struct {
	Expires time.Time
}

// NewReservation creates a default reservation
func NewReservation(path, id string, acl []gozk.ACL) Reservation {
	return &DefaultReservation{
		path: path,
		acl:  acl,
		id:   id,
	}
}

// Reserve will reserve an item with the id in the DefaultReservation for the given amount of
// time.
func (dr *DefaultReservation) Reserve(d time.Duration) error {
	log.Debugf("[Sync:Reservation] Attempting to reserve '%s' for %s...", dr.id, d)

	lockpath := constructPath(dr.path, dr.id)

	// Check if the reservation already exists
	b, _, err := zookeeper.Get(lockpath)
	if err == nil {

		// It exists, check if expired
		var expired time.Time
		expired.GobDecode(b)

		log.Debugf("[Sync:Reservation] Read existing node for '%s', expires at %s", dr.id, expired)

		if expired.Before(time.Now()) {
			log.Debugf("[Sync:Reservation] Deleting expired lock '%s'", dr.id)

			// It has expired, delete the node and get the reservation as usual
			zookeeper.Delete(lockpath, -1)
		} else {
			return ErrReserved
		}
	}

	expires := time.Now().Add(d)
	expiresBytes, err := expires.GobEncode()
	if err != nil {
		return err
	}

	for {
		_, err = zookeeper.Create(lockpath, expiresBytes, 0, dr.acl)
		if err == gozk.ErrNoNode {
			createParents(dr.path)
		} else if err == nil {
			break
		} else {
			log.Warnf("[Reservation] ZK error creating ephemeral lock node for reservation: %v", err)
			return err
		}
	}

	log.Debugf("[Sync:Reservation] Created lock node for '%s', expires at %s", dr.id, expires)

	return nil

}

// Release will release the reservation in the DefaultReservation
func (dr *DefaultReservation) Release() error {
	return zookeeper.Delete(constructPath(dr.path, dr.id), -1)
}

// AnonymousRelease will  release the reservation of the item with the given id and path
func AnonymousRelease(path, id string) error {
	return zookeeper.Delete(constructPath(path, id), -1)
}

func constructPath(path, id string) string {
	return fmt.Sprintf("%s/reserve-%s", path, id)
}
