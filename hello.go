// Hellofs implements a simple "hello world" file system.
package hellofs

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"golang.org/x/net/context"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

func Start() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("helloworld"),
		fuse.Subtype("hellofs"),
		fuse.LocalVolume(),
		fuse.VolumeName("Hello world!"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	realfs := NewRealFS()

	filesys := &FS{
		RemoteFS: realfs,
	}
	err = fs.Serve(c, filesys)
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

// FS implements the hello world file system.
type FS struct {
	RemoteFS *RealFS
}

func (fs *FS) Root() (fs.Node, error) {
	inode := &Inode{ino: 1, size: 0, nlink: 1, ctime: time.Now(), mtime: time.Now(), atime: time.Now()}
	root := NewDir(fs, inode.ino)
	fs.RemoteFS.Inodes[root.Ino] = inode
	fs.RemoteFS.Dirs[root.Ino] = make([]*Dentry, 0, 1024)

	return root, nil
}

// Dir implements both Node and Handle for the root directory.
type Dir struct {
	Fs  *FS
	Ino uint64
}

func NewDir(filesys *FS, ino uint64) *Dir {
	return &Dir{
		Fs:  filesys,
		Ino: ino,
	}
}

func (d *Dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	//type MkdirRequest struct {
	//	Header `json:"-"`
	//	Name   string
	//	Mode   os.FileMode
	//	// Umask of the request. Not supported on OS X.
	//	Umask os.FileMode
	//}
	//ino := d.Fs.GeneraterInode + 1
	//	node := NewInode(req.Name, ino, 0)
	inode := d.Fs.RemoteFS.CreateInode()
	d.Fs.RemoteFS.CreateDentry(d.Ino, inode.ino, req.Name, fuse.DT_Dir)
	child := NewDir(d.Fs, inode.ino)

	return child, nil
}

func (d *Dir) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	return nil
}

func (d *Dir) Forget() {
}

func (d *Dir) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	dir, ok := newDir.(*Dir)
	if !ok {
	}
	dentrys, ok := d.Fs.RemoteFS.Dirs[dir.Ino]
	if !ok {
	}
	for _, dentry := range dentrys {
		if dentry.Name == req.OldName {
			dentry.Name = req.NewName
			break
		}
	}

	return nil
}

func (d *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	inode := d.Fs.RemoteFS.CreateInode()
	//type CreateRequest struct {
	//	Header `json:"-"`
	//	Name   string
	//	Flags  OpenFlags
	//	Mode   os.FileMode
	//	// Umask of the request. Not supported on OS X.
	//	Umask os.FileMode
	//}
	d.Fs.RemoteFS.CreateDentry(d.Ino, inode.ino, req.Name, fuse.DT_File)
	child := NewFile(d.Fs, inode.ino)

	return child, child, nil
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	//type Attr struct {
	//	Valid time.Duration // how long Attr can be cached
	//
	//	Inode     uint64      // inode number
	//	Size      uint64      // size in bytes
	//	Blocks    uint64      // size in 512-byte units
	//	Atime     time.Time   // time of last access
	//	Mtime     time.Time   // time of last modification
	//	Ctime     time.Time   // time of last inode change
	//	Crtime    time.Time   // time of creation (OS X only)
	//	Mode      os.FileMode // file mode
	//	Nlink     uint32      // number of links (usually 1)
	//	Uid       uint32      // owner uid
	//	Gid       uint32      // group gid
	//	Rdev      uint32      // device numbers
	//	Flags     uint32      // chflags(2) flags (OS X only)
	//	BlockSize uint32      // preferred blocksize for filesystem I/O
	//	fmt.Println("dir: ", d.Ino, " call Attr")
	inode, ok := d.Fs.RemoteFS.Inodes[d.Ino]
	if ok {
		a.Inode = inode.ino
		a.Atime = inode.atime
		a.Mtime = inode.mtime
		a.Crtime = inode.ctime
		a.Uid = inode.uid
		a.Gid = inode.gid
		//		a.Mode = inode.mode
		a.Nlink = inode.nlink
		a.Mode = os.ModeDir | 0555
		a.Size = inode.size
	}
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	fmt.Println("dir: ", d.Ino, " call Lookup")
	dentrys := d.Fs.RemoteFS.Dirs[d.Ino]
	if len(dentrys) > 0 {
		for _, dentry := range dentrys {
			//			fmt.Println(reflect.TypeOf(node))
			if dentry.Name == name {
				if dentry.Type == fuse.DT_Dir {
					return &Dir{Fs: d.Fs, Ino: dentry.Inode}, nil
				}
				return &File{Fs: d.Fs, Ino: dentry.Inode}, nil
			}
		}
	}

	return nil, fuse.ENOENT
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fmt.Println("dir: ", d.Ino, " call ReadDirAll")
	var dirents []fuse.Dirent
	if dentrys, ok := d.Fs.RemoteFS.Dirs[d.Ino]; ok {
		if len(dentrys) > 0 {
			var d fuse.Dirent
			for _, dentry := range dentrys {
				if dentry.Type == fuse.DT_Dir {
					d = fuse.Dirent{Inode: dentry.Inode, Name: dentry.Name, Type: fuse.DT_Dir}
				} else {
					d = fuse.Dirent{Inode: dentry.Inode, Name: dentry.Name, Type: fuse.DT_File}
				}
				dirents = append(dirents, d)
			}
		}
	}

	return dirents, nil
}

func (d *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	//type RemoveRequest struct {
	//	Header `json:"-"`
	//	Name   string // name of the entry to remove
	//	Dir    bool   // is this rmdir?
	//
	fmt.Println("call remove: ")
	fmt.Println("req.Header: ", req.Header)
	fmt.Println("req.Name: ", req.Name)
	fmt.Println("req.Dir: ", req.Dir)
	dentrys := d.Fs.RemoteFS.Dirs[d.Ino]
	for index, dentry := range dentrys {
		if dentry.Name == req.Name {
			delete(d.Fs.RemoteFS.Inodes, dentry.Inode)
			d.Fs.RemoteFS.Dirs[d.Ino] = append(d.Fs.RemoteFS.Dirs[d.Ino][:index], d.Fs.RemoteFS.Dirs[d.Ino][index+1:]...)
			break
		}
	}

	return nil
}

// File implements both Node and Handle for the hello file.
type File struct {
	Fs  *FS
	Ino uint64
}

func NewFile(filesys *FS, ino uint64) *File {
	return &File{Fs: filesys, Ino: ino}
}

const greeting = "hello, world\n"

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	fmt.Println("File: ", f.Ino, " call Attr")
	inode, ok := f.Fs.RemoteFS.Inodes[f.Ino]
	if ok {
		fmt.Println("fill file attr")
		a.Inode = inode.ino
		a.Atime = inode.atime
		a.Mtime = inode.mtime
		a.Crtime = inode.ctime
		a.Uid = inode.uid
		a.Gid = inode.gid
		//		a.Mode = inode.mode
		a.Nlink = inode.nlink
		a.Mode = 0444
		a.Size = inode.size
	}

	return nil
}

func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(greeting), nil
	//fuse.ENOSYS not support
}

func (f *File) Forget() {
}

func (f *File) Flush(ctx context.Context, req *fuse.FlushRequest) (err error) {
	return fuse.ENOSYS
}

func (f *File) Fsync(ctx context.Context, req *fuse.FsyncRequest) (err error) {
	return fuse.ENOSYS
}

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	return f, fuse.ENOSYS
}
