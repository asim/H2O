package pkgmgr

import (
	"github.com/HailoOSS/provisioning-service/dao"
	"github.com/HailoOSS/provisioning-service/goget"
	"github.com/HailoOSS/provisioning-service/s3"
	"os"
)

var (
	defaultPkgMgr PkgMgr
)

type PkgMgr interface {
	Delete(*dao.ProvisionedService) error
	Download(*dao.ProvisionedService) (string, error)
	DownloadFile(string, string, string) (string, error)
	Exists(*dao.ProvisionedService) (bool, error)
	FileExists(string, string) (bool, error)
	IsDownloaded(*dao.ProvisionedService) (bool, string)
	VerifyBinary(*dao.ProvisionedService) error
	Setup() error
}

func init() {
	switch os.Getenv("H2O_PACKAGE_MANAGER") {
	case "goget":
		Init(goget.New())
	default:
		Init(s3.New())
	}
}

func Init(pm PkgMgr) {
	defaultPkgMgr = pm
}

func Delete(ps *dao.ProvisionedService) error {
	return defaultPkgMgr.Delete(ps)
}

func Download(ps *dao.ProvisionedService) (string, error) {
	return defaultPkgMgr.Download(ps)
}

func DownloadFile(prefix, rPath, lPath string) (string, error) {
	return defaultPkgMgr.DownloadFile(prefix, rPath, lPath)
}

func Exists(ps *dao.ProvisionedService) (bool, error) {
	return defaultPkgMgr.Exists(ps)
}

func FileExists(prefix, rPath string) (bool, error) {
	return defaultPkgMgr.FileExists(prefix, rPath)
}

func IsDownloaded(ps *dao.ProvisionedService) (bool, string) {
	return defaultPkgMgr.IsDownloaded(ps)
}

func VerifyBinary(ps *dao.ProvisionedService) error {
	return defaultPkgMgr.VerifyBinary(ps)
}

func Setup() {
	if err := defaultPkgMgr.Setup(); err != nil {
		panic("Provisioning service encountered during setup - " + err.Error())
	}
}
