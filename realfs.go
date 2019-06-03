package hellofs

import (
	"time"

	"bazil.org/fuse"
)

type RealFS struct {
	GeneraterInode uint64
	Dirs           map[uint64][]*Dentry
	Inodes         map[uint64]*Inode
}

func NewRealFS() *RealFS {
	return &RealFS{
		GeneraterInode: 1,
		Dirs:           make(map[uint64][]*Dentry),
		Inodes:         make(map[uint64]*Inode),
	}
}

func (r *RealFS) CreateInode() *Inode {
	r.GeneraterInode = r.GeneraterInode + 1
	inode := &Inode{
		ino:    r.GeneraterInode,
		size:   0,
		nlink:  1,
		uid:    0,
		gid:    0,
		ctime:  time.Now(),
		mtime:  time.Now(),
		atime:  time.Now(),
		target: []byte{},
	}
	r.Inodes[inode.ino] = inode

	return inode

}

func (r *RealFS) CreateDentry(dino, ino uint64, name string, t fuse.DirentType) *Dentry {
	d := &Dentry{
		Inode: ino,
		Name:  name,
		Type:  t,
	}
	r.Dirs[dino] = append(r.Dirs[dino], d)

	if t == fuse.DT_Dir {
		r.Dirs[ino] = make([]*Dentry, 0, 1024)
	}

	return d
}
