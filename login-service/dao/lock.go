package dao

import (
	"fmt"

	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/service/sync"
)

const (
	lockPath = "%s/%s"
)

type multiLock []sync.Lock

func (ml multiLock) Unlock() {
	for _, l := range ml {
		l.Unlock()
	}
}

// lockUser gets _and_ locks on the user, by unique ID plus all secondary ids
// will ALWAYS return a lock, which should be Unlock()ed
func lockUser(user *domain.User) (multiLock, error) {
	l, err := lockUserId(string(user.App), user.Uid)
	if err != nil {
		l.Unlock()
		return nil, err
	}

	ml := multiLock{l}
	for _, id := range user.Ids {
		l2, err := lockUserId(string(user.App), string(id))
		if err != nil {
			l2.Unlock()
			// unlock everything else
			ml.Unlock()
			return nil, err
		}

		ml = append(ml, l2)
	}

	return ml, nil
}

func lockUserId(namespace string, id string) (sync.Lock, error) {
	return sync.RegionLock([]byte(fmt.Sprintf(lockPath, namespace, id)))
}
