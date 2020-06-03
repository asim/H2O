package multiclient

import (
	"strings"
	"sync"

	"github.com/hailo-platform/H2O/go-hailo-lib/multierror"
	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
)

type Errors interface {
	error

	// IgnoreUid removes all errors for the given request uid(s)
	IgnoreUid(uids ...string) Errors
	// IgnoreService removes all errors for requests to the given service(s)
	IgnoreService(services ...string) Errors
	// IgnoreEndpoint removes all errors to a given service endpoint
	IgnoreEndpoint(service, endpoint string) Errors
	// IgnoreType removes all errors of the given type(s)
	IgnoreType(types ...string) Errors
	// IgnoreCode removes all errors with the given code(s)
	IgnoreCode(codes ...string) Errors
	// Errors returns all matching errors, mapped to their request uid
	Errors() map[string]errors.Error
	// Count returns a count of matching errors
	Count() int
	// AnyErrors returns whether there are any contained errors
	AnyErrors() bool
	// ForUid returns the error for a given request uid (or nil)
	ForUid(uid string) errors.Error
	// Suffix sets a dotted suffix to add to any returned errors
	Suffix(suffix string) Errors
	// Combined returns a platform error with the given suffix, scoped correctly (or nil)
	Combined() errors.Error
	// MultiError returns a combined MultiError
	MultiError() *multierror.MultiError
}

type reqErr struct {
	*client.Request
	err    errors.Error
	scoper Scoper
}

type errorsImpl struct {
	sync.RWMutex
	defaultScoper Scoper
	errs          map[string]reqErr // uid: reqErr
	suffix        string            // applied to errs at the right time, but kept for Combined() errors
}

// copyLessCap returns a new *errorsImpl, with the same default scoper, and pre-prepared (but empty) requestScopes and
// errs maps.
// Note that this requires an existing read lock.
func (e *errorsImpl) copyLessCap(lessCap int) *errorsImpl {
	newLen := len(e.errs) - lessCap
	if newLen < 0 {
		newLen = 0
	}
	return &errorsImpl{
		defaultScoper: e.defaultScoper,
		suffix:        e.suffix,
		errs:          make(map[string]reqErr, newLen),
	}
}

func (e *errorsImpl) set(uid string, req *client.Request, err errors.Error, scoper Scoper) *errorsImpl {
	e.Lock()
	defer e.Unlock()

	if e.errs == nil {
		e.errs = make(map[string]reqErr, 1)
	}
	e.errs[uid] = reqErr{
		Request: req,
		err:     err,
		scoper:  scoper,
	}
	return e
}

func (e *errorsImpl) IgnoreUid(uids ...string) Errors {
	if len(uids) == 0 {
		return e
	}
	uidsMap := stringsMap(uids...)
	e.RLock()
	defer e.RUnlock()
	result := e.copyLessCap(len(uids))
	for uid, v := range e.errs {
		if _, ok := uidsMap[uid]; !ok {
			result.errs[uid] = v
		}
	}
	return result
}

func (e *errorsImpl) IgnoreService(services ...string) Errors {
	if len(services) == 0 {
		return e
	}
	servicesMap := stringsMap(services...)
	e.RLock()
	defer e.RUnlock()
	result := e.copyLessCap(len(services))
	for uid, v := range e.errs {
		if _, ok := servicesMap[v.Service()]; !ok {
			result.errs[uid] = v
		}
	}
	return result
}

func (e *errorsImpl) IgnoreEndpoint(service, endpoint string) Errors {
	e.RLock()
	defer e.RUnlock()
	result := e.copyLessCap(1)
	for uid, v := range e.errs {
		if v.Service() != service && v.Endpoint() != endpoint {
			result.errs[uid] = v
		}
	}
	return result
}

func (e *errorsImpl) IgnoreType(types ...string) Errors {
	if len(types) == 0 {
		return e
	}
	typesMap := stringsMap(types...)
	e.RLock()
	defer e.RUnlock()
	result := e.copyLessCap(len(types))
	for uid, v := range e.errs {
		if _, ok := typesMap[v.err.Type()]; !ok {
			result.errs[uid] = v
		}
	}
	return result
}

func (e *errorsImpl) IgnoreCode(codes ...string) Errors {
	if len(codes) == 0 {
		return e
	}
	codesMap := stringsMap(codes...)
	e.RLock()
	defer e.RUnlock()
	result := e.copyLessCap(len(codes))
	for uid, v := range e.errs {
		if _, ok := codesMap[v.err.Code()]; !ok {
			result.errs[uid] = v
		}
	}
	return result
}

func (e *errorsImpl) Errors() map[string]errors.Error {
	e.RLock()
	defer e.RUnlock()
	if len(e.errs) < 1 {
		return nil
	}
	result := make(map[string]errors.Error, len(e.errs))
	for uid, v := range e.errs {
		result[uid] = v.err
	}
	return result
}

func (e *errorsImpl) Count() int {
	e.RLock()
	defer e.RUnlock()
	return len(e.errs)
}

func (e *errorsImpl) AnyErrors() bool {
	return e.Count() > 0
}

func (e *errorsImpl) ForUid(uid string) errors.Error {
	e.RLock()
	defer e.RUnlock()

	if re, ok := e.errs[uid]; ok {
		return re.err
	}

	return nil
}

func (e *errorsImpl) Suffix(suffix string) Errors {
	suffix = strings.TrimPrefix(suffix, ".") // Remove trailing dot
	if suffix == "" {
		return e
	}

	e.RLock()
	defer e.RUnlock()

	result := e.copyLessCap(0)
	if result.suffix != "" {
		result.suffix += "."
	}
	result.suffix += suffix
	for uid, re := range e.errs {
		result.errs[uid] = reqErr{
			Request: re.Request,
			err: scopedErr{
				errorType:   re.err.Type(),
				code:        re.err.Code() + "." + suffix,
				description: re.err.Description(),
				context:     re.err.Context(),
				httpCode:    re.err.HttpCode(),
				multiStack:  re.err.MultiStack(),
			},
			scoper: re.scoper}
	}
	return result
}

func (e *errorsImpl) Combined() errors.Error {
	e.RLock()
	defer e.RUnlock()

	switch len(e.errs) {
	case 0:
		return nil
	case 1:
		for _, re := range e.errs {
			return re.err
		}
		return nil
	default:
		// Figure out what Scoper to use for the error.
		// If each request has the same From, use that, otherwise, use the defaultScoper
		scoper, i := e.defaultScoper, 0
		for _, re := range e.errs {
			if re.scoper != nil && (i == 0 || scoper == nil || re.scoper == scoper) {
				scoper = re.scoper
			} else {
				scoper = e.defaultScoper
			}
			i++
		}

		context := ""
		if scoper != nil {
			context = scoper.Context()
		}
		if e.suffix != "" {
			if context != "" {
				context += "."
			}
			context += e.suffix
		}
		return errors.InternalServerError(context, e.multiError().Error())
	}
}

func (e *errorsImpl) multiError() *multierror.MultiError {
	me := multierror.New()
	for _, re := range e.errs {
		me.Add(re.err)
	}
	return me
}

func (e *errorsImpl) MultiError() *multierror.MultiError {
	e.RLock()
	defer e.RUnlock()
	return e.multiError()
}

func (e *errorsImpl) Error() string {
	return e.MultiError().Error()
}
