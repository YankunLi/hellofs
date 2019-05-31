// Hellofs implements a simple "hello world" file system.
package hellofs

import (
	"flag"
	"fmt"
	"log"
	"os"

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

func main() {
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

	filesys := &FS{
		GeneraterInode: 1,
		Inodes:         make(map[uint64]*Inode),
		Dirs:           make(map[uint64][]*Inode),
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

type Inode struct {
	Ino   uint64
	Size  uint64
	IType uint8
	Name  string
}

func NewInode(name string, ino uint64, t uint8) *Inode {
	return &Inode{Ino: ino, Name: name, Size: 5, IType: t}
}

// FS implements the hello world file system.
type FS struct {
	GeneraterInode uint64
	Inodes         map[uint64]*Inode
	Dirs           map[uint64][]*Inode
}

func (fs *FS) Root() (fs.Node, error) {
	node := &Inode{Ino: 1, Size: 512, IType: 0, Name: "root"}
	root := &Dir{Fs: fs, Ino: 1}
	fs.Inodes[root.Ino] = node
	fs.Dirs[root.Ino] = make([]*Inode, 0, 1024)

	return root, nil
}

// Dir implements both Node and Handle for the root directory.
type Dir struct {
	Fs  *FS
	Ino uint64
	//	Node *Inode
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
	ino := d.Fs.GeneraterInode + 1
	node := NewInode(req.Name, ino, 0)
	d.Fs.GeneraterInode = ino
	d.Fs.Dirs[d.Ino] = append(d.Fs.Dirs[d.Ino], node)
	d.Fs.Inodes[ino] = node
	d.Fs.Dirs[ino] = make([]*Inode, 0, 1024)
	child := NewDir(d.Fs, node.Ino)

	return child, nil
}

func (d *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	ino := d.Fs.GeneraterInode + 1
	node := NewInode(req.Name, ino, 1)
	d.Fs.GeneraterInode = ino
	d.Fs.Dirs[d.Ino] = append(d.Fs.Dirs[d.Ino], node)
	d.Fs.Inodes[ino] = node
	child := NewFile(d.Fs, node.Ino)

	return child, child, nil
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	fmt.Println("dir: ", d.Ino, " call Attr")
	node := d.Fs.Inodes[d.Ino]
	a.Inode = node.Ino
	a.Mode = os.ModeDir | 0555
	a.Size = node.Size

	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	fmt.Println("dir: ", d.Ino, " call Lookup")
	inodes := d.Fs.Dirs[d.Ino]
	if len(inodes) > 0 {
		for _, node := range inodes {
			//			fmt.Println(reflect.TypeOf(node))
			if node.Name == name {
				if node.IType == 0 {
					return &Dir{Fs: d.Fs, Ino: node.Ino}, nil
				}
				return &File{Fs: d.Fs, Ino: node.Ino}, nil
			}
		}
	}

	return nil, fuse.ENOENT
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fmt.Println("dir: ", d.Ino, " call ReadDirAll")
	var dirs []fuse.Dirent
	if inodes, ok := d.Fs.Dirs[d.Ino]; ok {
		if len(inodes) > 0 {
			var d fuse.Dirent
			for _, node := range inodes {
				if node.IType == 0 {
					d = fuse.Dirent{Inode: node.Ino, Name: node.Name, Type: fuse.DT_Dir}
				} else {
					d = fuse.Dirent{Inode: node.Ino, Name: node.Name, Type: fuse.DT_File}
				}
				dirs = append(dirs, d)
			}
		}
	}

	return dirs, nil
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
	inodes := d.Fs.Dirs[d.Ino]
	for index, node := range inodes {
		if node.Name == req.Name {
			delete(d.Fs.Inodes, node.Ino)
			d.Fs.Dirs[d.Ino] = append(d.Fs.Dirs[d.Ino][:index], d.Fs.Dirs[d.Ino][index+1:]...)
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
	node, ok := f.Fs.Inodes[f.Ino]
	if ok {
		a.Inode = node.Ino
		a.Mode = 0444
		a.Size = node.Size
	}

	return nil
}

func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(greeting), nil
}
