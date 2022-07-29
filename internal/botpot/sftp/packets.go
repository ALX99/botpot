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

// https://datatracker.ietf.org/doc/html/draft-ietf-secsh-filexfer-02#section-5
const (
	sshFileExferAttrSize        = 0x00000001
	sshFileExferAttrUIDGID      = 0x00000002
	sshFileExferAttrPermissions = 0x00000004
	sshFileExferAttrACModTime   = 0x00000008
	sshFileExferAttrExtended    = 0x80000000
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
	Filename  string // UTF-8
	Attrs     FileAttributes
	RequestID uint32
	PFlags    uint32
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface
func (p *Open) UnmarshalBinary(data []byte) error {
	var err error
	pb := newPacketBuffer(data)

	if p.RequestID, err = pb.readUint32(); err != nil {
		return err
	}
	if p.Filename, err = pb.readUTF8(); err != nil {
		return err
	}
	if p.PFlags, err = pb.readUint32(); err != nil {
		return err
	}

	return p.Attrs.UnmarshalBinary(pb.getRemainingBytes())
}

// Close SSH_FXP_CLOSE C->S
type Close struct {
	Handle    string
	RequestID uint32
}

// Read SSH_FXP_READ C->S
type Read struct {
	Handle    string
	Offset    uint64
	RequestID uint32
	Length    uint32
}

// Write SSH_FXP_WRITE C->S
type Write struct {
	Handle    string
	Data      string
	Offset    uint64
	RequestID uint32
}

// Lstat or SSH_FXP_LSTAT
type Lstat struct {
	Path      string // UTF-8
	RequestID uint32
	Flags     uint32
}

// FStat SSH_FXP_FSTAT C->S
type FStat struct {
	Handle    string
	RequestID uint32
	Flags     uint32
}

// SetStat SSH_FXP_SETSTAT C->S
type SetStat struct {
	Path      string // UTF-8
	Attrs     []byte // todo
	RequestID uint32
}

// FSetStat SSH_FXP_FSETSTAT C->S
type FSetStat struct {
	Handle    string
	Attrs     []byte // todo
	RequestID uint32
}

// OpenDir SSH_FXP_OPENDIR
type OpenDir struct {
	Path      string
	RequestID uint32
}

// ReadDir SSH_FXP_READDIR C->S
type ReadDir struct {
	Handle    string
	RequestID uint32
}

// Remove SSH_FXP_REMOVE C->S
type Remove struct {
	Filename  string // UTF-8
	RequestID uint32
}

// Mkdir SSH_FXP_MKDIR C->S
type Mkdir struct {
	Path      string
	Attrs     []byte
	RequestID uint32
}

// Rmdir SSH_FXP_RMDIR C->S
type Rmdir struct {
	Path      string // UTF-8
	RequestID uint32
}

// RealPath SSH_FXP_REALPATH C->S
type RealPath struct {
	OriginalPath string   // UTF-8
	ComposePath  []string // optional
	RequestID    uint32
	ControlByte  byte // optional
}

// Stat SSH_FXP_STAT
type Stat struct {
	Path      string // UTF-8
	RequestID uint32
	Flags     uint32
}

// Rename SSH_FXP_RENAME C->S
type Rename struct {
	OldPath   string // UTF-8
	NewPath   string // UTF-8
	RequestID uint32
	Flags     uint32
}

// Readlink SSH_FXP_READLINK C->S
type Readlink struct {
	Path      string // UTF-8
	RequestID uint32
}

// Link SSH_FXP_LINK C->S
type Link struct {
	NewLinkPath      string // UTF-8
	ExistingLinkPath string // UTF-8
	RequestID        uint32
	SymLink          bool
}

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
	RequestID uint32
}

// Unblock SSH_FXP_UNBLOCK
type Unblock struct {
	// Handle is returned by SSH_FXP_OPEN
	Handle string
	// Offset is the beggining of the byte-range to unlock
	Offset uint64
	// Number  of bytes to unlock
	Length    uint64
	RequestID uint32
}

// Status SSH_FXP_STATUS S->C
type Status struct {
	Message   string // ISO-10646 UTF-8 [RFC-2279]
	LangTag   string // RFC-1766
	RequestID uint32
	ErrorCode uint32
}

// Handle SSH_FXP_HANDLE S->C
type Handle struct {
	Handle    string
	RequestID uint32
}

// Data SSH_FXP_DATA S->C
type Data struct {
	Data      string
	RequestID uint32
	EOF       bool
}

// Name SSH_FXP_NAME S->C
type Name struct {
	Filename  []string // Count times
	Attrs     []byte   // Count times, Todo Attrs structure
	RequestID uint32
	Count     uint32
	EOL       bool // Optional
}

// Attrs SSH_FXP_ATTRS
type Attrs struct {
	Attrs     []byte //  Todo Attrs structure
	RequestID uint32
}

// Extended SSH_FXP_EXTENDED
type Extended struct {
	ExtendedRequest string
	ExtensionData   []byte
	RequestID       uint32
}

// ExtendedReply SSH_FXP_EXTENDED_REPLY
type ExtendedReply struct {
	ExtensionData []byte
	RequestID     uint32
}

// FileAttributes https://datatracker.ietf.org/doc/html/draft-ietf-secsh-filexfer-02#section-5
type FileAttributes struct {
	ExtendedType  []string
	ExtendedData  []string
	Size          uint64
	Flags         uint32
	Permissions   uint32
	Atime         uint32
	Mtime         uint32
	ExtendedCount uint32
	UID           uint32
	GID           uint32
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface
func (fa *FileAttributes) UnmarshalBinary(data []byte) error {
	pb := newPacketBuffer(data)
	var err error

	if fa.Flags, err = pb.readUint32(); err != nil {
		return err
	}

	if fa.Flags&sshFileExferAttrSize != 0 {
		if fa.Size, err = pb.readUint64(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFileExferAttrUIDGID != 0 {
		if fa.UID, err = pb.readUint32(); err != nil {
			return err
		}
		if fa.GID, err = pb.readUint32(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFileExferAttrPermissions != 0 {
		if fa.Permissions, err = pb.readUint32(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFileExferAttrACModTime != 0 {
		if fa.Atime, err = pb.readUint32(); err != nil {
			return err
		}
		if fa.Mtime, err = pb.readUint32(); err != nil {
			return err
		}
	}

	if fa.Flags&sshFileExferAttrExtended == 0 {
		return nil
	}

	if fa.ExtendedCount, err = pb.readUint32(); err != nil {
		return err
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
	data   []byte
	length uint32
	pType  byte
}
