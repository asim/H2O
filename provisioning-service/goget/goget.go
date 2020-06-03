package goget

import (
	"bytes"
	"fmt"
	"github.com/hailo-platform/H2O/provisioning-service/dao"
	"github.com/hailo-platform/H2O/provisioning-service/process"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var (
	envPath = "PATH=/opt/boxen/homebrew/bin:$PATH"
	gitUrl  = "github.com/hailo-platform/H2O"
)

type GoGetMgr struct {
	WorkDir string
	GoPath  string
	GitUrl  string
}

func New() *GoGetMgr {
	tmpDir := path.Join(os.Getenv("HOME"), "/tmp/hailo")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		panic("Error creating provisioning service working directory - " + err.Error())
	}

	return &GoGetMgr{
		WorkDir: tmpDir,
		GoPath:  tmpDir,
		GitUrl:  gitUrl,
	}
}

func cp(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

func run(dir, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()

	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (g *GoGetMgr) gitCp(repo, remotePath, localPath string) error {
	// Copy from repo into place
	filePath := path.Join(g.WorkDir, "src", g.GitUrl, repo, remotePath)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("%s does not exist in repo", filePath)
	}

	if err := cp(filePath, localPath); err != nil {
		return fmt.Errorf("failed to copy %s to %s, %v", filePath, localPath, err)
	}

	return nil
}

func (g *GoGetMgr) gitUpdate(repo string) error {
	// make sure folder exists
	dir := path.Join(g.WorkDir, "src", g.GitUrl)
	if err := os.MkdirAll(dir, os.ModeDir|os.ModePerm); err != nil {
		return err
	}

	localRepo := path.Join(dir, repo)
	if _, err := os.Stat(localRepo); os.IsNotExist(err) {
		// git clone the repo
		remoteRepo := fmt.Sprintf("https://%s/%s", g.GitUrl, repo)
		if err := run("", "git", "clone", remoteRepo, path.Join(dir, repo)); err != nil {
			return err
		}
	} else {
		// git pull
		if err := run(path.Join(dir, repo), "git", "pull", "-u"); err != nil {
			return err
		}
	}

	return nil
}

func (g *GoGetMgr) goGet(serviceName string, exePath string) error {
	exeDir := filepath.Dir(exePath)
	service := strings.Split(serviceName, ".")
	servicePath := fmt.Sprintf("%s/%s-service", g.GitUrl, service[len(service)-1])

	os.Setenv("GOPATH", g.GoPath)
	os.Setenv("GOBIN", exeDir)

	if err := run("", "go", "get", "-d", "-u", servicePath); err != nil {
		return err
	}

	if err := run("", "go", "build", "-o", exePath, servicePath); err != nil {
		return err
	}

	return nil
}

func (g *GoGetMgr) Setup() error {
	return nil
}

func (g *GoGetMgr) Exists(ps *dao.ProvisionedService) (bool, error) {
	return true, nil
}

func (g *GoGetMgr) FileExists(repo, remotePath string) (bool, error) {
	return true, nil
}

func (g *GoGetMgr) Download(ps *dao.ProvisionedService) (string, error) {
	dst := process.ExePath(ps)

	// make sure folder exists
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, os.ModeDir|os.ModePerm); err != nil {
		return dst, err
	}

	if err := g.goGet(ps.ServiceName, dst); err != nil {
		return dst, err
	}

	if err := os.Chmod(dst, 0777); err != nil {
		return dst, err
	}

	return dst, nil
}

func (g *GoGetMgr) DownloadFile(repo, remotePath, localPath string) (string, error) {
	if err := g.gitUpdate(repo); err != nil {
		return localPath, err
	}

	// make sure folder exists
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, os.ModeDir|os.ModePerm); err != nil {
		return localPath, err
	}

	// copy the file
	if err := g.gitCp(repo, remotePath, localPath); err != nil {
		return localPath, err
	}

	if err := os.Chmod(localPath, 0755); err != nil {
		return localPath, err
	}

	return localPath, nil
}

// IsDownloaded will check if we have already downloaded this binary/JAR to
// the local filesystem, returning the full path to the file
func (g *GoGetMgr) IsDownloaded(ps *dao.ProvisionedService) (bool, string) {
	dst := process.ExePath(ps)
	if _, err := os.Stat(dst); err != nil {
		return false, dst
	}

	return true, dst
}

// Delete removes a downloaded file, incase of errors copying
func (g *GoGetMgr) Delete(ps *dao.ProvisionedService) error {
	if ok, dst := g.IsDownloaded(ps); ok {
		if err := os.Remove(fmt.Sprintf("%s.md5", dst)); err != nil {
			return err
		}

		return os.Remove(dst)
	}

	return nil
}

func (g *GoGetMgr) VerifyBinary(ps *dao.ProvisionedService) error {
	return nil
}
