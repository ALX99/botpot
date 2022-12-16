package sftp

import (
	"encoding/binary"
	"errors"
)

const (
	sshFXPInit     = 1
	sshFXPVersion  = 2
	sshFXPOpen     = 3
	sshFXPClose    = 4
	sshFXPRead     = 5
	sshFXPWrite    = 6
	sshFXPLstat    = 7
	sshFXPFstat    = 8
	sshFXPSetStat  = 9
	sshFXPFsetStat = 10
	sshFXPOpenDir  = 11
	sshFXPReadDir  = 12
	sshFXPRemove   = 13
	sshFXPMkdir    = 14
	sshFXPRmdir    = 15
	sshFXPRealPath = 16
	sshFXPStat     = 17
	sshFXPRename   = 18
	sshFXPReadlink = 19
	sshFXPLink     = 21
	sshFXPBlock    = 22
	sshFXPUnblock  = 23

	sshFXPStatus = 101
	sshFXPHandle = 102
	sshFXPData   = 103
	sshFXPName   = 104
	sshFXPAttrs  = 105

	sshFXPExtended      = 200
	sshFXPExtendedReply = 201
)

// https://datatracker.ietf.org/doc/html/draft-ietf-secsh-filexfer-13#section-7.1
const (
	sshFilexferAttrSize             = 0x00000001
	sshFilexferAttrPermissions      = 0x00000004
	sshFilexferAttrAccessTime       = 0x00000008
	sshFilexferAttrCreateTime       = 0x00000010
	sshFilexferAttrModifyTime       = 0x00000020
	sshFilexferAttrACL              = 0x00000040
	sshFilexferAttrOwnergroup       = 0x00000080
	sshFilexferAttrSubsecondTimes   = 0x00000100
	sshFilexferAttrBits             = 0x00000200
	sshFilexferAttrAllocationSize   = 0x00000400
	sshFilexferAttrTextHint         = 0x00000800
	sshFilexferAttrMimeType         = 0x00001000
	sshFilexferAttrLinkCount        = 0x00002000
	sshFilexferAttrUntranslatedName = 0x00004000
	sshFilexferAttrCTime            = 0x00008000
	sshFilexferAttrExtended         = 0x80000000
)

// Init SSH_FXP_INIT C->S
type Init struct {
	Version uint32
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface
func (p *Init) UnmarshalBinary(data []byte) error {
	if len(data) != 4 {
		return errors.New("expected 4 byte data")
	}
	p.Version = binary.BigEndian.Uint32(data)
	return nil
}

// Version SSH_FXP_VERSION S->C
type Version struct {
	ExtensionPair [][]byte
	Version       uint32
}

// Open SSH_FXP_OPEN C->S
type Open struct {
	Filename      string // UTF-8
	Attrs         FileAttributes
	Flags         uint32
	DesiredAccess uint32
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface
func (p *Open) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	if p.Filename, err = pb.readUTF8(); err != nil {
		return err
	}
	if p.DesiredAccess, err = pb.readUint32(); err != nil {
		return err
	}
	if p.Flags, err = pb.readUint32(); err != nil {
		return err
	}

	b := pb.getRemainingBytes()
	if len(b) > 0 {
		return p.Attrs.UnmarshalBinary(b)
	}
	return nil
}

// Close SSH_FXP_CLOSE C->S
type Close struct {
	Handle string
}

func (p *Close) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	// TODO not sure this one is working
	p.Handle, err = pb.readUTF8()
	if err != nil {
		return err
	}

	return nil
}

// Read SSH_FXP_READ C->S
type Read struct {
	Handle string
	Offset uint64
	Length uint32
}

func (p *Read) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	// TODO not sure this one is working
	p.Handle, err = pb.readUTF8()
	if err != nil {
		return err
	}

	p.Offset, err = pb.readUint64()
	if err != nil {
		return err
	}

	p.Length, err = pb.readUint32()
	if err != nil {
		return err
	}

	return nil
}

// Write SSH_FXP_WRITE C->S
type Write struct {
	Handle string
	Data   string
	Offset uint64
}

func (p *Write) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	// TODO not sure this one is working
	p.Handle, err = pb.readUTF8()
	if err != nil {
		return err
	}

	p.Offset, err = pb.readUint64()
	if err != nil {
		return err
	}

	p.Data, err = pb.readUTF8()
	if err != nil {
		return err
	}

	return nil
}

// Remove SSH_FXP_REMOVE C->S
type Remove struct {
	Filename string // UTF-8
}

func (p *Remove) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	// TODO not sure this one is working
	p.Filename, err = pb.readUTF8()
	if err != nil {
		return err
	}

	return nil
}

// Rename SSH_FXP_RENAME C->S
type Rename struct {
	OldPath string // UTF-8
	NewPath string // UTF-8
	Flags   uint32
}

func (p *Rename) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	// TODO not sure this one is working
	p.OldPath, err = pb.readUTF8()
	if err != nil {
		return err
	}

	// TODO not sure this one is working
	p.NewPath, err = pb.readUTF8()
	if err != nil {
		return err
	}

	return nil
}

// Mkdir SSH_FXP_MKDIR C->S
type Mkdir struct {
	Path  string
	Attrs FileAttributes
}

func (p *Mkdir) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	p.Path, err = pb.readUTF8()
	if err != nil {
		return err
	}

	b := pb.getRemainingBytes()
	if len(b) > 0 {
		return p.Attrs.UnmarshalBinary(b)
	}
	return nil
}

// Rmdir SSH_FXP_RMDIR C->S
type Rmdir struct {
	Path string // UTF-8
}

func (p *Rmdir) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	p.Path, err = pb.readUTF8()
	if err != nil {
		return err
	}

	return nil
}

// OpenDir SSH_FXP_OPENDIR
type OpenDir struct {
	Path string
}

func (p *OpenDir) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	p.Path, err = pb.readUTF8()
	if err != nil {
		return err
	}

	return nil
}

// ReadDir SSH_FXP_READDIR C->S
type ReadDir struct {
	Handle string
}

func (p *ReadDir) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	p.Handle, err = pb.readUTF8()
	if err != nil {
		return err
	}

	return nil
}

// Stat SSH_FXP_STAT
type Stat struct {
	Path string // UTF-8
}

func (p *Stat) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	p.Path, err = pb.readUTF8()
	if err != nil {
		return err
	}

	return nil
}

// Lstat or SSH_FXP_LSTAT
type Lstat struct {
	Path string // UTF-8
}

func (p *Lstat) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	p.Path, err = pb.readUTF8()
	if err != nil {
		return err
	}

	return nil
}

// FStat SSH_FXP_FSTAT C->S
type FStat struct {
	Handle string
}

func (p *FStat) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	p.Handle, err = pb.readUTF8()
	if err != nil {
		return err
	}

	return nil
}

// SetStat SSH_FXP_SETSTAT C->S
type SetStat struct {
	Path  string // UTF-8
	Attrs []byte // todo
}

func (p *SetStat) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// FSetStat SSH_FXP_FSETSTAT C->S
type FSetStat struct {
	Handle string
	Attrs  []byte // todo
}

func (p *FSetStat) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// RealPath SSH_FXP_REALPATH C->S
type RealPath struct {
	OriginalPath string   // UTF-8
	ComposePath  []string // optional
	ControlByte  byte     // optional
}

func (p *RealPath) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// Readlink SSH_FXP_READLINK C->S
type Readlink struct {
	Path string // UTF-8
}

func (p *Readlink) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// Link SSH_FXP_LINK C->S
type Link struct {
	NewLinkPath      string // UTF-8
	ExistingLinkPath string // UTF-8
	SymLink          bool
}

func (p *Link) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// Block SSH_FXP_BLOCK
type Block struct {
	// Handle is returned by SSH_FXP_OPEN
	Handle string
	// Offset is the beggining of the byte-range to lock
	Offset uint64
	// Number  of bytes to lock
	Length uint64
	// A bitmask of SSH_FXF_BLOCK_* values
	ULockMast uint32
}

func (p *Block) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// Unblock SSH_FXP_UNBLOCK
type Unblock struct {
	// Handle is returned by SSH_FXP_OPEN
	Handle string
	// Offset is the beggining of the byte-range to unlock
	Offset uint64
	// Number  of bytes to unlock
	Length uint64
}

func (p *Unblock) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// Status SSH_FXP_STATUS S->C
type Status struct {
	Message   string // ISO-10646 UTF-8 [RFC-2279]
	LangTag   string // RFC-1766
	ErrorCode uint32
}

func (p *Status) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// Handle SSH_FXP_HANDLE S->C
type Handle struct {
	Handle string
}

func (p *Handle) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// Data SSH_FXP_DATA S->C
type Data struct {
	Data string
	EOF  bool
}

func (p *Data) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// Name SSH_FXP_NAME S->C
type Name struct {
	Filename []string // Count times
	Attrs    []byte   // Count times, Todo Attrs structure
	Count    uint32
	EOL      bool // Optional
}

func (p *Name) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// Attrs SSH_FXP_ATTRS
type Attrs struct {
	Attrs []byte //  Todo Attrs structure
}

func (p *Attrs) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// Extended SSH_FXP_EXTENDED
type Extended struct {
	ExtendedRequest string
	ExtensionData   []byte
}

func (p *Extended) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// ExtendedReply SSH_FXP_EXTENDED_REPLY
type ExtendedReply struct {
	ExtensionData []byte
}

func (p *ExtendedReply) UnmarshalBinary(data []byte) error { return errors.New("not implemented") }

// FileAttributes https://datatracker.ietf.org/doc/html/draft-ietf-secsh-filexfer-13#section-7
type FileAttributes struct {
	Owner              string
	UntranslatedName   string
	MimeType           string
	ACL                string
	Group              string
	ExtendedData       []string
	ExtendedType       []string
	Size               uint64
	AllocationSize     uint64
	Atime              int64
	CreateTime         int64
	CTime              int64
	Permissions        uint32
	CTimeNSeconds      uint32
	CreateTimeNSeconds uint32
	AttribBits         uint32
	AttribBitsValid    uint32
	AtimeNSeconds      uint32
	LinkCount          uint32
	Flags              uint32
	ExtendedCount      uint32
	TextHint           byte
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface
func (fa *FileAttributes) UnmarshalBinary(data []byte) error {
	pb := newPacketBuffer(data)
	var err error

	if fa.Flags, err = pb.readUint32(); err != nil {
		return err
	}

	if fa.Flags&sshFilexferAttrSize != 0 {
		if fa.Size, err = pb.readUint64(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFilexferAttrAllocationSize != 0 {
		if fa.AllocationSize, err = pb.readUint64(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFilexferAttrOwnergroup != 0 {
		if fa.Owner, err = pb.readUTF8(); err != nil {
			return err
		}
		if fa.Group, err = pb.readUTF8(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFilexferAttrPermissions != 0 {
		if fa.Permissions, err = pb.readUint32(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFilexferAttrAccessTime != 0 {
		if fa.Atime, err = pb.readInt64(); err != nil {
			return err
		}
		if fa.Flags&sshFilexferAttrSubsecondTimes != 0 {
			if fa.AtimeNSeconds, err = pb.readUint32(); err != nil {
				return err
			}
		}
	}

	if fa.Flags&sshFilexferAttrCreateTime != 0 {
		if fa.CreateTime, err = pb.readInt64(); err != nil {
			return err
		}
		if fa.Flags&sshFilexferAttrSubsecondTimes != 0 {
			if fa.CreateTimeNSeconds, err = pb.readUint32(); err != nil {
				return err
			}
		}
	}

	if fa.Flags&sshFilexferAttrCTime != 0 {
		if fa.CTime, err = pb.readInt64(); err != nil {
			return err
		}
		if fa.Flags&sshFilexferAttrSubsecondTimes != 0 {
			if fa.CTimeNSeconds, err = pb.readUint32(); err != nil {
				return err
			}
		}
	}

	if fa.Flags&sshFilexferAttrACL != 0 {
		if fa.ACL, err = pb.readUTF8(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFilexferAttrBits != 0 {
		if fa.AttribBits, err = pb.readUint32(); err != nil {
			return err
		}
		if fa.AttribBitsValid, err = pb.readUint32(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFilexferAttrTextHint != 0 {
		if fa.TextHint, err = pb.readUint8(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFilexferAttrMimeType != 0 {
		if fa.MimeType, err = pb.readUTF8(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFilexferAttrLinkCount != 0 {
		if fa.LinkCount, err = pb.readUint32(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFilexferAttrUntranslatedName != 0 {
		if fa.UntranslatedName, err = pb.readUTF8(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFilexferAttrExtended != 0 {
		if fa.ExtendedCount, err = pb.readUint32(); err != nil {
			return err
		}
	}

	fa.ExtendedType = make([]string, fa.ExtendedCount)
	fa.ExtendedData = make([]string, fa.ExtendedCount)

	for i := uint32(0); i < fa.ExtendedCount; i++ {
		if fa.ExtendedType[i], err = pb.readUTF8(); err != nil {
			return err
		}
		if fa.ExtendedData[i], err = pb.readUTF8(); err != nil {
			return err
		}
	}

	return nil
}

// packet represents an SFTP packet
// https://datatracker.ietf.org/doc/html/draft-ietf-secsh-filexfer-13#section-4
type packet struct {
	data      []byte
	length    uint32
	pType     byte
	requestID uint32 // Not set for INIT and VERSION
}
