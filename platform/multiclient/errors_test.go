package multiclient

import (
	"testing"

	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/errors"
	ptesting "github.com/HailoOSS/platform/testing"
)

func TestErrorsImplSuite(t *testing.T) {
	ptesting.RunSuite(t, new(errorsImplSuite))
}

type errorsImplSuite struct {
	ptesting.Suite
	rawErrs map[string]errors.Error
	errs    *errorsImpl
}

func (suite *errorsImplSuite) SetupTest() {
	suite.Suite.SetupTest()
	suite.rawErrs = map[string]errors.Error{
		"uid1": errors.InternalServerError("com.foo.uid1", "uid1"),
		"uid2": errors.InternalServerError("com.foo.uid2", "uid2"),
		"uid3": errors.InternalServerError("com.foo.uid2", "uid3"), // Same service uid as uid2
	}
	suite.errs = &errorsImpl{}
	for uid, err := range suite.rawErrs {
		req, reqErr := client.NewJsonRequest(err.Code(), err.Description(), nil)
		if reqErr != nil {
			panic(reqErr)
		}
		suite.errs.set(uid, req, err, nil)
	}
}

func (suite *errorsImplSuite) TearDownSuite() {
	suite.Suite.TearDownTest()
	suite.rawErrs = nil
	suite.errs = nil
}

func (suite *errorsImplSuite) TestBasic() {
	errs := suite.errs
	err1 := suite.rawErrs["uid1"]
	err2 := suite.rawErrs["uid2"]
	err3 := suite.rawErrs["uid3"]

	suite.Assertions.Equal(err1, errs.ForUid("uid1"))
	suite.Assertions.Equal(err2, errs.ForUid("uid2"))
	suite.Assertions.Equal(err3, errs.ForUid("uid3"))
	suite.Assertions.True(errs.MultiError().AnyErrors())
	suite.Assertions.Equal(3, errs.Count())
}

func (suite *errorsImplSuite) TestIgnoreUid() {
	errs := suite.errs
	err2 := suite.rawErrs["uid2"]

	errs = errs.IgnoreUid("uid1").(*errorsImpl)
	suite.Assertions.Nil(errs.ForUid("uid1"))
	suite.Assertions.Equal(err2, errs.ForUid("uid2"))
	suite.Assertions.True(errs.MultiError().AnyErrors())
	suite.Assertions.Equal(2, errs.Count())

	errs = errs.IgnoreUid("uid2", "uid3").(*errorsImpl)
	suite.Assertions.Nil(errs.ForUid("uid1"))
	suite.Assertions.Nil(errs.ForUid("uid2"))
	suite.Assertions.False(errs.MultiError().AnyErrors())
	suite.Assertions.Equal(0, errs.Count())
}

func (suite *errorsImplSuite) TestIgnoreService() {
	errs := suite.errs
	err2 := suite.rawErrs["uid2"]
	err3 := suite.rawErrs["uid3"]

	errs = errs.IgnoreService("com.foo.uid1").(*errorsImpl)
	suite.Assertions.Nil(errs.ForUid("uid1"))
	suite.Assertions.Equal(err2, errs.ForUid("uid2"))
	suite.Assertions.Equal(err3, errs.ForUid("uid3"))
	suite.Assertions.True(errs.MultiError().AnyErrors())
	suite.Assertions.Equal(2, errs.Count())

	errs = errs.IgnoreService("com.foo.uid2").(*errorsImpl) // uid2 and uid3 have the same service
	suite.Assertions.Nil(errs.ForUid("uid1"))
	suite.Assertions.Nil(errs.ForUid("uid2"))
	suite.Assertions.Nil(errs.ForUid("uid3"))
	suite.Assertions.False(errs.MultiError().AnyErrors())
	suite.Assertions.Equal(0, errs.Count())
}

func (suite *errorsImplSuite) TestIgnoreEndpoint() {
	errs := suite.errs
	err2 := suite.rawErrs["uid2"]
	err3 := suite.rawErrs["uid3"]

	errs = errs.IgnoreEndpoint("com.foo.uid1", "uid1").(*errorsImpl)
	suite.Assertions.Nil(errs.ForUid("uid1"))
	suite.Assertions.Equal(err2, errs.ForUid("uid2"))
	suite.Assertions.Equal(err3, errs.ForUid("uid3"))
	suite.Assertions.True(errs.MultiError().AnyErrors())
	suite.Assertions.Equal(2, errs.Count())

	errs = errs.IgnoreEndpoint("com.foo.uid1", "uid10").(*errorsImpl) // Doesn't exist
	suite.Assertions.Nil(errs.ForUid("uid1"))
	suite.Assertions.Equal(err2, errs.ForUid("uid2"))
	suite.Assertions.Equal(err3, errs.ForUid("uid3"))
	suite.Assertions.True(errs.MultiError().AnyErrors())
	suite.Assertions.Equal(2, errs.Count())
}

func (suite *errorsImplSuite) TestIgnoreType() {
	errs := suite.errs

	errs = errs.IgnoreType(errors.ErrorInternalServer).(*errorsImpl)
	suite.Assertions.Nil(errs.ForUid("uid1"))
	suite.Assertions.Nil(errs.ForUid("uid2"))
	suite.Assertions.Nil(errs.ForUid("uid3"))
	suite.Assertions.False(errs.MultiError().AnyErrors())
	suite.Assertions.Equal(0, errs.Count())
}

func (suite *errorsImplSuite) TestIgnoreCode() {
	errs := suite.errs
	err2 := suite.rawErrs["uid2"]
	err3 := suite.rawErrs["uid3"]

	errs = errs.IgnoreCode("com.foo.uid1").(*errorsImpl)
	suite.Assertions.Nil(errs.ForUid("uid1"))
	suite.Assertions.Equal(err2, errs.ForUid("uid2"))
	suite.Assertions.Equal(err3, errs.ForUid("uid3"))
	suite.Assertions.True(errs.MultiError().AnyErrors())
	suite.Assertions.Equal(2, errs.Count())

	errs = errs.IgnoreCode("com.foo.uid2", "com.foouid3").(*errorsImpl)
	suite.Assertions.Nil(errs.ForUid("uid1"))
	suite.Assertions.Nil(errs.ForUid("uid2"))
	suite.Assertions.Nil(errs.ForUid("uid3"))
	suite.Assertions.False(errs.MultiError().AnyErrors())
	suite.Assertions.Equal(0, errs.Count())
}

func (suite *errorsImplSuite) TestErrors() {
	errs := suite.errs.Errors()
	err1 := suite.rawErrs["uid1"]
	err2 := suite.rawErrs["uid2"]
	err3 := suite.rawErrs["uid3"]

	suite.Assertions.Equal(err1, errs["uid1"])
	suite.Assertions.Equal(err2, errs["uid2"])
	suite.Assertions.Equal(err3, errs["uid3"])
}

func (suite *errorsImplSuite) TestForUid() {
	errs := suite.errs
	err1 := suite.rawErrs["uid1"]
	err2 := suite.rawErrs["uid2"]
	err3 := suite.rawErrs["uid3"]

	suite.Assertions.Equal(err1, errs.ForUid("uid1"))
	suite.Assertions.Equal(err2, errs.ForUid("uid2"))
	suite.Assertions.Equal(err3, errs.ForUid("uid3"))
}

func (suite *errorsImplSuite) TestSuffix() {
	errs := suite.errs.Suffix("foo")
	suite.Assertions.Equal("com.foo.uid1.foo", errs.ForUid("uid1").Code())
	suite.Assertions.Equal("com.foo.uid2.foo", errs.ForUid("uid2").Code())
	suite.Assertions.Equal("com.foo.uid2.foo", errs.ForUid("uid3").Code())

	// Empty suffix
	errs = suite.errs.Suffix("")
	suite.Assertions.Equal("com.foo.uid1", errs.ForUid("uid1").Code())
	suite.Assertions.Equal("com.foo.uid2", errs.ForUid("uid2").Code())
	suite.Assertions.Equal("com.foo.uid2", errs.ForUid("uid3").Code())

	// Redundant dot suffix
	errs = suite.errs.Suffix(".bar")
	suite.Assertions.Equal("com.foo.uid1.bar", errs.ForUid("uid1").Code())
	suite.Assertions.Equal("com.foo.uid2.bar", errs.ForUid("uid2").Code())
	suite.Assertions.Equal("com.foo.uid2.bar", errs.ForUid("uid3").Code())
}

func (suite *errorsImplSuite) TestEmpty() {
	errs := &errorsImpl{}
	suite.Assertions.Equal(0, errs.Count())
	suite.Assertions.Nil(errs.ForUid("foobar"))
	suite.Assertions.Nil(errs.Combined())
	suite.Assertions.False(errs.MultiError().AnyErrors())
	suite.Assertions.NotPanics(func() {
		errs.IgnoreUid("foo").
			IgnoreService("foo").
			IgnoreEndpoint("foo", "foo").
			IgnoreType("foo").
			IgnoreCode("foo").
			Suffix("foo")
	})
	suite.Assertions.Nil(errs.Errors())
}
