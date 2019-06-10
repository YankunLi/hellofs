package hellofs

import (
	"os"
	"time"

	"bazil.org/fuse"
	"github.com/tiglabs/containerfs/proto"
)

type Inode struct {
	ino    uint64
	size   uint64
	nlink  uint32
	uid    uint32
	gid    uint32
	ctime  time.Time
	mtime  time.Time
	atime  time.Time
	mode   os.FileMode
	target []byte
}

type Dentry struct {
	Inode uint64
	Name  string
	Type  fuse.DirentType
}

func NewDentry(ino uint64, name string, t fuse.DirentType) *Dentry {
	return &Dentry{
		Inode: ino,
		Name:  name,
		Type:  t,
	}
}

//func NewInode(info *proto.InodeInfo) *Inode {
//	inode := new(Inode)
//	inode.fill(info)
//	return inode
//}

//func (s *Super) InodeGet(ino uint64) (*Inode, error) {
//	inode := s.ic.Get(ino)
//	if inode != nil {
//		//log.LogDebugf("InodeCache hit: inode(%v)", inode)
//		return inode, nil
//	}
//
//	info, err := s.mw.InodeGet_ll(ino)
//	if err != nil || info == nil {
//		log.LogErrorf("InodeGet: ino(%v) err(%v) info(%v)", ino, err, info)
//		if err != nil {
//			return nil, ParseError(err)
//		} else {
//			return nil, fuse.ENOENT
//		}
//	}
//	inode = NewInode(info)
//	s.ic.Put(inode)
//	return inode, nil
//}
//
//func (inode *Inode) String() string {
//	return fmt.Sprintf("ino(%v) mode(%v) size(%v) nlink(%v) uid(%v) gid(%v) exp(%v) mtime(%v) target(%v)", inode.ino, inode.mode, inode.size, inode.nlink, inode.uid, inode.gid, time.Unix(0, inode.expiration).Format(LogTimeFormat), inode.mtime, inode.target)
//}
//
func (inode *Inode) setattr(req *fuse.SetattrRequest) (valid uint32) {
	if req.Valid.Mode() {
		inode.mode = req.Mode
		valid |= proto.AttrMode
	}

	if req.Valid.Uid() {
		inode.uid = req.Uid
		valid |= proto.AttrUid
	}

	if req.Valid.Gid() {
		inode.gid = req.Gid
		valid |= proto.AttrGid
	}
	return
}

//
////func (inode *Inode) fill(info *proto.InodeInfo) {
////	inode.ino = info.Inode
////	inode.size = info.Size
////	inode.nlink = info.Nlink
////	inode.uid = info.Uid
////	inode.gid = info.Gid
////	inode.ctime = info.CreateTime
////	inode.atime = info.AccessTime
////	inode.mtime = info.ModifyTime
////	inode.target = info.Target
////	inode.mode = proto.OsMode(info.Mode)
////}
//
func (inode *Inode) fillAttr(attr *fuse.Attr) {
	//attr.Valid = AttrValidDuration
	attr.Nlink = inode.nlink
	attr.Inode = inode.ino
	attr.Mode = inode.mode
	attr.Size = inode.size
	attr.Blocks = attr.Size >> 9 // In 512 bytes
	attr.Atime = inode.atime
	attr.Ctime = inode.ctime
	attr.Mtime = inode.mtime
	attr.BlockSize = 4096
	attr.Uid = inode.uid
	attr.Gid = inode.gid
}
