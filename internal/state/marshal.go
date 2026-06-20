package state

import (
	"bytes"
	"fmt"
	"strconv"
)

func marshalRegistry(registry Registry) []byte {
	var b bytes.Buffer
	if registry.Default != "" {
		b.WriteString("default = ")
		b.WriteString(strconv.Quote(registry.Default))
		b.WriteString("\n")
		if len(registry.Sources) > 0 {
			b.WriteString("\n")
		}
	}

	for i, source := range registry.Sources {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[[source]]\n")
		b.WriteString("id = ")
		b.WriteString(strconv.Quote(source.ID))
		b.WriteString("\n")
		b.WriteString("path = ")
		b.WriteString(strconv.Quote(source.Path))
		b.WriteString("\n")
		fmt.Fprintf(&b, "enabled = %t\n", source.Enabled)
	}
	for _, copyRecord := range registry.Copies {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[[copy]]\n")
		writeStringField(&b, "source", copyRecord.Source)
		writeStringField(&b, "context", copyRecord.Context)
		writeStringField(&b, "package", copyRecord.Package)
		writeStringField(&b, "path", copyRecord.Path)
		writeStringField(&b, "target", copyRecord.Target)
		writeStringField(&b, "sourceChecksum", copyRecord.SourceChecksum)
		writeStringField(&b, "targetChecksum", copyRecord.TargetChecksum)
		writeStringField(&b, "targetMode", copyRecord.TargetMode)
	}

	return b.Bytes()
}

func writeStringField(b *bytes.Buffer, name string, value string) {
	b.WriteString(name)
	b.WriteString(" = ")
	b.WriteString(strconv.Quote(value))
	b.WriteString("\n")
}
