package dav

import (
	"github.com/G-Node/gogs/pkg/context"
	"golang.org/x/net/webdav"
	"os"
	"fmt"
	"net/http"
	"github.com/G-Node/gogs/models"
	"regexp"
	"github.com/G-Node/git-module"
	"time"
	"io"
)

var (
	RE_GETRNAME = regexp.MustCompile(".+/(.+)/_dav")
	RE_GETROWN  = regexp.MustCompile("./(.+)/.+/_dav")
	RE_GETFPATH = regexp.MustCompile("/_dav/(.+)")
)

func Dav(c *context.Context, handler *webdav.Handler) {
	if checkPerms(c) != nil {
		c.WriteHeader(http.StatusUnauthorized)
		return
	}
	handler.ServeHTTP(c.Resp, c.Req.Request)
	return
}

// GinFS implements webdav (it implements webdav.Habdler) read only access to a repository
type GinFS struct {
	BasePath string
}

// Just return an error. -> Read Only
func (fs *GinFS) Mkdir(name string, perm os.FileMode) error {
	return fmt.Errorf("Mkdir not implemented for read only gin FS")
}

// Just return an error. -> Read Only
func (fs *GinFS) RemoveAll(name string) error {
	return fmt.Errorf("Remove not implemented for read only gin FS")
}

// Just return an error. -> Read Only
func (fs *GinFS) Rename(oldName, newName string) error {
	return fmt.Errorf("Rename not implemented for read only gin FS")
}

func (fs *GinFS) OpenFile(name string, flag int, perm os.FileMode) (webdav.File, error) {
	//todo: catch all the errors
	rname, _ := getRName(name)
	oname, _ := getOName(name)
	path, _ := getFPath(name)
	grepo, _ := git.OpenRepository(fmt.Sprintf("%s/%s/%s.git", oname, rname))
	com, _ := grepo.GetBranchCommit("master")
	tree, _ := com.SubTree(path)
	trentry, _ := com.GetTreeEntryByPath(path)
	return GinFile{trentry: trentry, tree: tree}, nil
}

func (fs GinFS) Stat(name string) (os.FileInfo, error) {
	f, err := fs.OpenFile(name, 0, 0)
	if err != nil {
		return nil, err
	}
	return f.Stat()
}

type GinFile struct {
	tree      *git.Tree
	trentry   *git.TreeEntry
	dirrcount int
}

func (f GinFile) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("Write to GinFile not implemented (read only)")
}

func (f GinFile) Close() error {
	return nil
}

func (f GinFile) Read(p []byte) (n int, err error) {
	if f.trentry.Type != git.OBJECT_BLOB {
		return 0, fmt.Errorf("not a blob")
	}
	data, err := f.trentry.Blob().Data()
	if err != nil{
		return 0, err
	}
	// todo: annex
	return data.Read(p)
}

func (f GinFile) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (f GinFile) Readdir(count int) ([]os.FileInfo, error) {
	ents, err := f.tree.ListEntries()
	if err != nil {
		return nil, err
	}
	if count <= 0 {
		infos := make([]os.FileInfo, len(ents))
		for c, ent := range ents {
			finfo, err := GinFile{trentry: ent}.Stat()
			if err != nil {
				return nil, err
			}
			infos[c] = finfo
		}
		return infos, nil
	} else {

	}
}

func (f GinFile) Stat() (os.FileInfo, error) {
	return GinFinfo{f.trentry}, nil
}

type GinFinfo struct {
	*git.TreeEntry
}

func (i GinFinfo) Mode() os.FileMode {
	return 0
}

func (i GinFinfo) ModTime() time.Time {
	return time.Now()
}

func (i GinFinfo) Sys() interface{} {
	return nil
}


func checkPerms(c *context.Context) error {
	return nil
}

func getRepo(path string) (*models.Repository, error) {
	oID, err := getROwnerID(path)
	if err != nil {
		return nil, err
	}

	rname, err := getRName(path)
	if err != nil {
		return nil, err
	}

	return models.GetRepositoryByName(oID, rname)
}

func getRName(path string) (string, error) {
	name := RE_GETRNAME.FindStringSubmatch(path)
	if len(name) > 1 {
		return name[1], nil
	}
	return "", fmt.Errorf("Could not determine repo name")
}

func getOName(path string) (string, error) {
	name := RE_GETROWN.FindStringSubmatch(path)
	if len(name) > 1 {
		return name[1], nil
	}
	return "", fmt.Errorf("Could not determine repo owner")
}

func getFPath(path string) (string, error) {
	name := RE_GETFPATH.FindStringSubmatch(path)
	if len(name) > 1 {
		return name[1], nil
	}
	return "", fmt.Errorf("Could not determine file path")
}

func getROwnerID(path string) (int64, error) {
	name := RE_GETROWN.FindStringSubmatch(path)
	if len(name) > 1 {
		models.GetUserByName(name[1])
	}
	return -100, fmt.Errorf("Could not determine repo owner")
}