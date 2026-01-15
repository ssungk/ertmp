package transport

// Protocol constants
const (
	RTMPVersion                = 3
	HandshakeSize              = 1536
	DefaultChunkSize           = 128
	MaxChunkSize               = 0xFFFFFF   // 16777215
	ChunkSizeMsgMask           = 0x7FFFFFFF // SetChunkSize message: MSB must be 0 (31-bit value)
	IOBufferSize               = 8192       // 8KB
	DefaultWindowAckSize       = 2500000
	DefaultPeerBandwidth       = 2500000
	ExtendedTimestampThreshold = 0xFFFFFF
)

// Message Type IDs
const (
	MsgTypeSetChunkSize     = 0x01
	MsgTypeAbort            = 0x02
	MsgTypeAcknowledgement  = 0x03
	MsgTypeUserControl      = 0x04
	MsgTypeWindowAckSize    = 0x05
	MsgTypeSetPeerBW        = 0x06
	MsgTypeAudio            = 0x08
	MsgTypeVideo            = 0x09
	MsgTypeAMF3Data         = 0x0F
	MsgTypeAMF3SharedObject = 0x10
	MsgTypeAMF3Command      = 0x11
	MsgTypeAMF0Data         = 0x12
	MsgTypeAMF0SharedObject = 0x13
	MsgTypeAMF0Command      = 0x14
	MsgTypeAggregate        = 0x16
)

// Chunk Stream IDs
const (
	ChunkStreamProtocol = 2
	ChunkStreamCommand  = 3
	ChunkStreamAudio    = 4
	ChunkStreamVideo    = 5
	ChunkStreamData     = 6
)

// User Control Event Types
const (
	UserControlStreamBegin      = 0x00
	UserControlStreamEOF        = 0x01
	UserControlStreamDry        = 0x02
	UserControlSetBufferLen     = 0x03
	UserControlStreamIsRecorded = 0x04
	UserControlPingRequest      = 0x06
	UserControlPingResponse     = 0x07
)

// Bandwidth Limit Types
const (
	LimitTypeHard    = 0
	LimitTypeSoft    = 1
	LimitTypeDynamic = 2
)

// Standard Audio Codecs (RTMP 1.0)
const (
	AudioCodecLinearPCM    = 0x00
	AudioCodecADPCM        = 0x01
	AudioCodecMP3          = 0x02
	AudioCodecLinearPCMLE  = 0x03
	AudioCodecNellymoser16 = 0x04
	AudioCodecNellymoser8  = 0x05
	AudioCodecNellymoser   = 0x06
	AudioCodecALaw         = 0x07
	AudioCodecMuLaw        = 0x08
	AudioCodecAAC          = 0x0A
	AudioCodecSpeex        = 0x0B
	AudioCodecMP38kHz      = 0x0E
	AudioCodecDeviceSpec   = 0x0F
)

// Standard Video Codecs (RTMP 1.0)
const (
	VideoCodecJPEG     = 0x01
	VideoCodecH263     = 0x02
	VideoCodecScreenV1 = 0x03
	VideoCodecOn2VP6   = 0x04
	VideoCodecOn2VP6A  = 0x05
	VideoCodecScreenV2 = 0x06
	VideoCodecH264     = 0x07
)

// Video Frame Types
const (
	VideoFrameTypeKey        = 0x01
	VideoFrameTypeInter      = 0x02
	VideoFrameTypeDisposable = 0x03
	VideoFrameTypeGenerated  = 0x04
	VideoFrameTypeInfo       = 0x05
)

// AVC Packet Types
const (
	AVCPacketTypeSequenceHeader = 0x00
	AVCPacketTypeNALU           = 0x01
	AVCPacketTypeEndOfSequence  = 0x02
)

// Chunk Message Header Format Types
const (
	FmtType0 = 0 // 전체 헤더
	FmtType1 = 1 // 동일한 스트림 ID
	FmtType2 = 2 // 동일한 길이와 스트림 ID
	FmtType3 = 3 // 헤더 없음
)
