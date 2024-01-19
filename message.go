package hl7

import (
	"bufio"
	"bytes"
	"io"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

// Constants describing possible message boundaries.
const (
	CR = '\r'   // Carriage return
	LF = '\n'   // Line feed
	FF = '\f'   // Form feed
	NB = '\x00' // Null byte
)

// Message is used to describe the parsed message.
type Message struct {
	segments   map[string][]Segment
	reader     *bufio.Reader
	lock       sync.Mutex
	fieldSep   byte
	compSep    byte
	subCompSep byte
	repeat     byte
	escape     byte
}

// Parse is used to parse the segments within the message so that they can be
// queried and iterated. This is a different paradigm from the ReadSegment
// method, which parses the segments as-needed.
func (m *Message) Parse() error {
	m.segments = map[string][]Segment{}

	for {
		segment, err := m.ReadSegment()

		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		stype := segment.Type()
		m.segments[stype] = append(m.segments[stype], segment)
	}
	return nil
}

// Returns the first segment with a matching SubComponent Name / ID
// MSH, PID, etc.
func (m *Message) GetSegment(id string) ([]Segment, error) {
	return m.segments[id], nil
}

// ReadSegment is used to "read" the next segment from the message.
func (m *Message) ReadSegment() (Segment, error) {
	var buf []byte

	m.lock.Lock()

	for {
		b, err := m.reader.ReadByte()

		if err == io.EOF {
			break
		}

		// Skip all line feeds and character returns while we haven't started saving
		// bytes to the byte slice. This helps cope with messages that have a lot of
		// extra whitespace in them.
		if len(buf) == 0 && unicode.IsSpace(rune(b)) {
			continue
		}

		if b == CR || b == LF {
			break
		}

		buf = append(buf, b)
	}

	m.lock.Unlock()

	if len(buf) == 0 {
		return Segment{}, io.EOF
	}
	return newSegment(m.fieldSep, m.compSep, m.subCompSep, m.repeat, m.escape, buf), nil
}

// Find gets a value from a message using location syntax
// finds the first occurence of the segment and first of repeating fields
// if the loc is not valid an error is returned
func (m *Message) Find(loc string) (string, error) {
	return m.Get(NewLocation(loc))
}

// FindAll gets all values from a message using location syntax
// finds all occurences of the segments and all repeating fields
// if the loc is not valid an error is returned
// func (m *Message) FindAll(loc string) ([]string, error) {
// 	return m.GetAll(NewLocation(loc))
// }

// Get returns the first value specified by the Location
func (m *Message) Get(l *Location) (string, error) {
	// if l.Segment == "" {
	// 	return string(m.Value), nil
	// }
	seg, err := m.GetSegment(l.Segment)
	if err != nil {
		return "", err
	}
	sc, _ := seg[0].GetSubComponent(l.FieldSeq, 0, l.Comp, l.SubComp)
	return sc.String(), err
}

// GetAll returns all values specified by the Location
// func (m *Message) GetAll(l *Location) ([]string, error) {
// 	vals := []string{}
// 	if l.Segment == "" {
// 		vals = append(vals, string(m.Value))
// 		return vals, nil
// 	}
// 	segs, err := m.AllSegments(l.Segment)
// 	if err != nil {
// 		return vals, err
// 	}
// 	for _, s := range segs {
// 		vs, err := s.GetAll(l)
// 		if err != nil {
// 			return vals, err
// 		}
// 		vals = append(vals, vs...)
// 	}
// 	return vals, nil
// }

// Unmarshal fills a structure from an HL7 message
// It will panic if interface{} is not a pointer to a struct
// Unmarshal will decode the entire message before trying to set values
// it will set the first matching segment / first matching field
// repeating segments and fields is not well suited to this
// for the moment all unmarshal target fields must be strings
func (m *Message) Unmarshal(it interface{}) error {
	st := reflect.ValueOf(it).Elem()
	stt := st.Type()
	for i := 0; i < st.NumField(); i++ {
		fld := stt.Field(i)
		r := fld.Tag.Get("hl7")
		if r != "" {
			if val, _ := m.Find(r); val != "" {
				if st.Field(i).CanSet() {
					// TODO support fields other than string
					//fldT := st.Field(i).Type()
					st.Field(i).SetString(strings.TrimSpace(val))
				}
			}
		}
	}

	return nil
}

// NewMessage takes a byte slice and returns a Message that is ready to use.
func NewMessage(data []byte) (*Message, error) {
	// The message must have at least 8 bytes in order to catch all of the
	// character definitions in the header.
	if len(data) < 8 {
		return nil, io.EOF
	}
	reader := bytes.NewBuffer(data)

	m := Message{
		reader:     bufio.NewReader(reader),
		fieldSep:   data[3],
		compSep:    data[4],
		repeat:     data[5],
		escape:     data[6],
		subCompSep: data[7],
	}
	return &m, nil
}
