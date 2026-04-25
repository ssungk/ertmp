package amf

// AMF0 Type Markers
const (
	numberMarker      = 0x00
	booleanMarker     = 0x01
	stringMarker      = 0x02
	objectMarker      = 0x03
	movieClipMarker   = 0x04 // reserved
	nullMarker        = 0x05
	undefinedMarker   = 0x06
	referenceMarker   = 0x07 // not implemented
	ecmaArrayMarker   = 0x08
	objectEndMarker   = 0x09
	strictArrayMarker = 0x0A
	dateMarker        = 0x0B
	longStringMarker  = 0x0C
	unsupportedMarker = 0x0D // not implemented
	recordSetMarker   = 0x0E // reserved
	xmlDocumentMarker = 0x0F // not implemented
	typedObjectMarker = 0x10 // not implemented
	avmPlusMarker     = 0x11 // not implemented (AMF3)
)

