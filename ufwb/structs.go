//go:ignore generate stringer -type Endian,Display,LengthUnit,Order
//go:generate getter -type Grammar,GrammarRef,Custom,String,Structure,StructRef,Binary,Number,Offset,Script
// TODO Consider moving this into a separate package, so that the parser can't use the unexported fields (and forced to go via Getters, which "do the right thing" wrt extending and defaults.
package ufwb

import (
	"bramp.net/dsector/toerr"
	"fmt"
	"io"
	"strconv"
)

const (
	Black = Colour(0x000000)
	White = Colour(0xffffff)
)

type Colour uint32
type Bool int8 // tri-state bool unset, false, true.

type Expression interface {
	fmt.Stringer
}

type ConstExpression int64

func (e ConstExpression) String() string {
	return fmt.Sprintf("ConstExpression(%d)", int64(e))
}

// StringExpression needs to be evaluated
type StringExpression string

func (e StringExpression) String() string {
	return fmt.Sprintf("StringExpression(%q)", string(e))
}

func NewExpression(expr string) Expression {
	if expr == "" {
		// Empty string means it wasn't set
		return nil
	}

	// Try a number
	if i, err := strconv.ParseInt(expr, 0, 64); err == nil {
		return ConstExpression(i)
	}

	// TODO if expr == unlimited

	return StringExpression(expr)
}

// No other value is allowed
const (
	UnknownBool Bool = iota
	False
	True
)

func (b Bool) bool() bool {
	switch b {
	case False:
		return false
	case True:
		return true
	}
	panic("Unknown bool state")
}

func boolOf(b bool) Bool {
	if b {
		return True
	}
	return False
}

type Endian int

const (
	UnknownEndian Endian = 0
	DynamicEndian        = 1
	BigEndian            = 2
	LittleEndian         = 3
)

type Display int

const (
	UnknownDisplay Display = iota
	BinaryDisplay
	DecDisplay
	HexDisplay
)

func (d Display) Base() int {
	switch d {
	case HexDisplay:
		return 16
	case DecDisplay:
		return 10
	case BinaryDisplay:
		return 2
	case UnknownDisplay:
		return 0
	}
	panic(fmt.Sprintf("unknown base %d", d))
}

type LengthUnit int

const (
	UnknownLengthUnit LengthUnit = iota
	BitLengthUnit
	ByteLengthUnit
)

type Order int

const (
	UnknownOrder Order = iota
	FixedOrder         // TODO Check this is the right name
	VariableOrder
)

type Reader interface {
	// Read from file and return a Value.
	// The Read method must leave the file offset at Value.Offset + Value.Len // TODO Enforce this!
	// Read should return what has been parsed, and any error encountered.
	// If no bytes could be read, then <nil, io.EOF> is returned.
	Read(decoder *Decoder) (*Value, error)
}

type Formatter interface {
	// Format returns the display string for this Element.
	Format(file io.ReaderAt, value *Value) (string, error)
}

type Updatable interface {
	// Updates/validates the Element
	update(u *Ufwb, parent *Structure, errs *toerr.Errors)
}

type Derivable interface {
	DeriveFrom(parent Element) error
}

type Repeatable interface {
	RepeatMin() Expression
	RepeatMax() Expression
}

// ElementId holds the the basic identifer for a Element
type ElementId interface {
	Id() int
	Name() string
	Description() string

	IdString() string
}

type Lengthable interface {
	Length() Expression
	LengthUnit() LengthUnit
}

type Element interface {
	ElementId

	Reader
	Lengthable
	Repeatable
	Updatable
	Derivable
	Formatter

	// TODO Add Colourful here
}

type Colourful struct {
	fillColour   *Colour `default:"White" dereference:"true" parent:"false"`
	strokeColour *Colour `default:"Black" dereference:"true" parent:"false"`
}

type Ufwb struct {
	Xml *XmlUfwb

	Version string
	Grammar *Grammar

	Elements map[string]Element
	Scripts  map[string]*Script
}

// Base is what all Elements implement
type Base struct {
	elemType    string `parent:"false" derives:"false" setter:"false"` // This field is only for debug printing
	id          int    `parent:"false" derives:"false"`
	name        string `parent:"false" derives:"false"`
	description string `parent:"false" derives:"false"`
}

func (b *Base) Id() int {
	return b.id
}
func (b *Base) Name() string {
	return b.name
}
func (b *Base) Description() string {
	return b.description
}

func (b *Base) GetBase() *Base {
	return b
}

func (b *Base) IdString() string {
	if b == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s<%02d %s>", b.elemType, b.id, b.name)
}

type Repeats struct {
	repeatMin Expression `default:"ConstExpression(1)" parent:"false"`
	repeatMax Expression `default:"ConstExpression(1)" parent:"false"`
}

type Grammar struct {
	Xml *XmlGrammar

	Base
	Repeats

	Author   string
	Ext      string
	Email    string
	Complete Bool
	Uti      string

	Start    *Structure
	Scripts  []*Script
	Elements []Element
}

type Structure struct {
	Xml *XmlStructure

	Base
	Repeats
	Colourful

	derives *Structure
	parent  *Structure

	length       Expression `parent:"false"`
	lengthUnit   LengthUnit `default:"ByteLengthUnit"`
	lengthOffset Expression

	endian   Endian `default:"LittleEndian"`
	signed   Bool   `default:"True"`
	encoding string `default:"\"UTF-8\""`

	order Order `default:"FixedOrder"`

	display Display `default:"DecDisplay"`

	elements []Element `parent:"false"`

	/*
		Encoding  string `xml:"encoding,attr,omitempty" ufwb:"encoding"`
		Alignment string `xml:"alignment,attr,omitempty"` // ??

		Floating   string `xml:"floating,attr,omitempty"` // ??
		ConsistsOf string `xml:"consists-of,attr,omitempty" ufwb:"id"`

		Repeat    string `xml:"repeat,attr,omitempty" ufwb:"id"`
		RepeatMin string `xml:"repeatmin,attr,omitempty" ufwb:"ref"`
		RepeatMax string `xml:"repeatmax,attr,omitempty" ufwb:"ref"`

		ValueExpression string `xml:"valueexpression,attr,omitempty"`
		Debug           string `xml:"debug,attr,omitempty" ufwb:"bool"`
		Disabled        string `xml:"disabled,attr,omitempty" ufwb:"bool"`
	*/
}

type GrammarRef struct {
	Xml *XmlGrammarRef

	Base
	Repeats
	derives *GrammarRef

	uti      string
	filename string
	disabled Bool

	grammar *Grammar // TODO Actually load this!
}

type Custom struct {
	Xml *XmlCustom

	Base
	Colourful

	derives *Custom

	length     Expression `parent:"false"`
	lengthUnit LengthUnit `default:"ByteLengthUnit"`

	script *Script
}

type StructRef struct {
	Xml *XmlStructRef

	Base
	Repeats
	Colourful
	disabled Bool

	derives *StructRef

	structure *Structure
}

type String struct {
	Xml *XmlString

	Base
	Repeats
	Colourful

	derives *String
	parent  *Structure

	typ string // TODO Convert to "StringType" // "zero-terminated", "fixed-length", "pascal", "delimiter-terminated"

	length     Expression `parent:"false"`
	lengthUnit LengthUnit `default:"ByteLengthUnit"`

	encoding string `default:"\"UTF-8\""`

	delimiter byte // Used when typ is "delimiter-terminated" or "zero-terminated"

	mustMatch Bool `default:"True"`
	values    []*FixedStringValue
}

type Binary struct {
	Xml *XmlBinary

	Base
	Repeats
	Colourful

	derives *Binary
	parent  *Structure

	length     Expression `parent:"false"`
	lengthUnit LengthUnit `default:"ByteLengthUnit"`

	//unused     Bool // TODO
	//disabled   Bool

	mustMatch Bool `default:"True"`
	values    []*FixedBinaryValue
}

type Number struct {
	Xml *XmlNumber

	Base
	Repeats
	Colourful

	derives *Number
	parent  *Structure

	Type       string     // TODO Convert to Type
	length     Expression `parent:"false"`
	lengthUnit LengthUnit `default:"ByteLengthUnit"`

	endian Endian `default:"LittleEndian"`
	signed Bool   `default:"True"`

	display Display `default:"DecDisplay"`

	// TODO Handle the below fields:
	valueExpression string

	minVal string // TODO This should be a int
	maxVal string

	mustMatch Bool `default:"True"`
	values    []*FixedValue
	masks     []*Mask
}

// TODO Support parsing the Offsets
type Offset struct {
	Xml *XmlOffset

	Base
	Repeats
	Colourful

	derives *Offset
	parent  *Structure

	length     Expression `parent:"false"`
	lengthUnit LengthUnit `default:"ByteLengthUnit"`

	endian Endian `default:"LittleEndian"`

	display Display `default:"DecDisplay"`

	relativeTo          ElementId
	followNullReference Bool
	references          ElementId
	referencedSize      ElementId
	additional          string
}

type Script struct {
	Xml *XmlScriptElement

	Base
	Repeats

	derives *Script

	//Disabled bool

	XmlScript *XmlScript
	typ       string // TODO Change to a enum
	language  string // TODO Change to a enum

	text string
}

type Mask struct {
	Xml *XmlMask

	name        string
	value       uint64 // The mask
	description string `parent:"false" derives:"false"`

	values []*FixedValue
}

// TODO FixedValue is for what? Numbers? Rename to FixedNumberValue
type FixedValue struct {
	Xml *XmlFixedValue

	name  string
	value interface{}

	description string
}

type FixedBinaryValue struct {
	Xml *XmlFixedValue

	name  string
	value []byte

	description string
}

type FixedStringValue struct {
	Xml *XmlFixedValue

	name  string
	value string

	description string
}

// Padding is a pseudo Element created to represent unspecified regions in a file.
type Padding struct {
	Base
}

func (*Padding) Length() Expression {
	return nil
}

func (*Padding) LengthUnit() LengthUnit {
	return ByteLengthUnit
}

func (*Padding) RepeatMax() Expression {
	return ConstExpression(1)
}

func (*Padding) RepeatMin() Expression {
	return ConstExpression(1)
}

func (*Padding) update(*Ufwb, *Structure, *toerr.Errors) {
	// Do nothing
}

type Elements []Element

func (e Elements) Find(name string) (int, Element) {
	for i, element := range e {
		if element.Name() == name {
			return i, element
		}
	}
	return -1, nil
}

// ElementByName returns a child element with this name, or nil
func (s *Structure) ElementByName(name string) Element {
	if _, e := Elements(s.elements).Find(name); e != nil {
		return e
	}

	if s.derives != nil {
		return s.derives.ElementByName(name)
	}

	return nil
}

func (s *Structure) Signed() bool {
	// TODO Move this to be auto generated
	if s.signed != UnknownBool {
		return s.signed.bool()
	}
	if s.derives != nil {
		return s.derives.Signed()
	}
	if s.parent != nil {
		return s.parent.Signed()
	}
	return true
}

func (s *Structure) SetSigned(signed bool) {
	s.signed = boolOf(signed)
}

func (n *Number) Signed() bool {
	if n.signed != UnknownBool {
		return n.signed.bool()
	}
	if n.derives != nil {
		return n.derives.Signed()
	}
	if n.parent != nil {
		return n.parent.Signed()
	}
	return true
}

func (n *Number) SetSigned(signed bool) {
	n.signed = boolOf(signed)
}

func (s *StructRef) Length() Expression {
	return s.Structure().Length()
}
func (s *StructRef) LengthUnit() LengthUnit {
	return s.Structure().LengthUnit()
}

func (g *GrammarRef) Length() Expression {
	return g.Grammar().Length()
}
func (g *GrammarRef) LengthUnit() LengthUnit {
	return g.Grammar().LengthUnit()
}

func (g *Grammar) Length() Expression {
	return nil // Always unset
}

func (g *Grammar) LengthUnit() LengthUnit {
	return ByteLengthUnit // Always unset
}

func (s *Script) Length() Expression {
	// ScriptElements have no form, thus no length
	return ConstExpression(0)
}

func (s *Script) LengthUnit() LengthUnit {
	return ByteLengthUnit
}

func (*Custom) RepeatMin() Expression {
	return ConstExpression(1)
}

func (*Custom) RepeatMax() Expression {
	return ConstExpression(1)
}
