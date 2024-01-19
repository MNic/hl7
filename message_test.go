package hl7

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {
	type PIDMessage struct {
		Id        string `hl7:"PID.0.0"`
		FirstName string `hl7:"PID.5.1"`
		LastName  string `hl7:"PID.5.0"`
		Ehr       string `hl7:"MSH.2.0"`
	}
	hl7text := `MSH|^~\&|CERNER||PriorityHealth||||ORU^R01|Q479004375T431430612|P|2.3|
	PID|||001677980||SMITH^CURTIS||19680219|M||||||||||929645156318|123456789|
	PD1||||1234567890^LAST^FIRST^M^^^^^NPI|
	OBR|1|341856649^HNAM_ORDERID|000002006326002362|648088^Basic Metabolic Panel|||20061122151600|||||||||1620^Hooker^Robert^L||||||20061122154733|||F|||||||||||20061122140000|
	OBX|1|NM|GLU^Glucose Lvl|59|mg/dL|65-99^65^99|L||||F|||20061122154733|	
	`
	pid_message := PIDMessage{}
	msg, err := NewMessage([]byte(hl7text))
	if err != nil {
		log.Fatalln("failed message creation")
	}
	msg.Parse()
	// log for visibility
	// seg, _ := msg.GetSegment("PID")
	// fn, _ := seg[0].GetSubComponent(5, 0, 1, 0)
	// t.Error(fmt.Printf("fn: %v", fn.String()))
	// continue original test
	msg.Unmarshal(&pid_message)
	assert.Equal(t, "PID", pid_message.Id)
	assert.Equal(t, "CURTIS", pid_message.FirstName)
	assert.Equal(t, "SMITH", pid_message.LastName)
	assert.Equal(t, "CERNER", pid_message.Ehr)
}

func TestGetSegment(t *testing.T) {
	file, err := os.ReadFile("./testdata/msg.hl7")
	if err != nil {
		log.Fatalf("Failed to read file")
	}
	reader := NewReader(bytes.NewReader(file))
	msg, err := reader.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read message")
	}
	msg.Parse()
	seg, err := msg.GetSegment(("PID"))
	if err != nil {
		log.Fatalf("Unable to read segment")
	}
	LastName, _ := seg[0].GetSubComponent(5, 0, 0, 0)
	FirstName, _ := seg[0].GetSubComponent(5, 0, 1, 0)
	assert.Equal(t, "John", FirstName.String())
	assert.Equal(t, "Jones", LastName.String())
}

func TestNewMessage(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *Message
		wantErr bool
	}{
		{"Empty (nil)", []byte(nil), nil, true},
		{"Empty (not nil)", []byte{}, nil, true},
		{"Too short", []byte(`MSH|^~\`), nil, true},
		{
			"Minimal example",
			[]byte(`MSH|^~\&`),
			&Message{
				reader:     bufio.NewReader(bytes.NewBuffer([]byte(`MSH|^~\&`))),
				fieldSep:   '|',
				compSep:    '^',
				subCompSep: '&',
				repeat:     '~',
				escape:     '\\',
			},
			false,
		},
		{
			"Custom separators",
			[]byte("MSH....."),
			&Message{
				reader:     bufio.NewReader(bytes.NewBuffer([]byte("MSH....."))),
				fieldSep:   '.',
				compSep:    '.',
				subCompSep: '.',
				repeat:     '.',
				escape:     '.',
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMessage(tt.data)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMessageParse(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		counts map[string]int
	}{
		{
			"one segment",
			[]byte("MSH|^~\\&"),
			map[string]int{"MSH": 1},
		},
		{
			"two segments",
			[]byte("MSH|^~\\&\rMSH|^~\\&"),
			map[string]int{"MSH": 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, _ := NewMessage(tt.data)
			msg.Parse()

			// Verify that the parsed segments match up with what we expect (this can
			// catch unexpected segments showing up).
			for stype, segments := range msg.segments {
				wantCount := tt.counts[stype]
				assert.Equal(t, wantCount, len(segments))
			}

			// Verify that all the counts we expect match with what was parsed (this
			// can catch missing segments).
			for stype, wantCount := range tt.counts {
				segments := msg.segments[stype]
				assert.Equal(t, wantCount, len(segments))
			}
		})
	}
}

func TestMessageReadSegment(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		count int
	}{
		{"one segment", []byte("MSH|^~\\&"), 1},
		{"two segments", []byte("MSH|^~\\&\rMSH|^~\\&"), 2},
		{"two segments, extra whitespace", []byte("MSH|^~\\&\r\nMSH|^~\\&"), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, _ := NewMessage(tt.data)

			for i := 0; i < tt.count; i++ {
				_, err := msg.ReadSegment()

				assert.Nil(t, err)
			}
			_, err := msg.ReadSegment()

			assert.Error(t, err)
		})
	}
}
