package stdlib

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"

	"github.com/gabrielcampellounb/gopherflow/internal/runtime"
)

var decimal = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)

func source(args []runtime.Value) (runtime.Value, error) {
	path, ok := args[0].(runtime.String)
	if !ok {
		return nil, fmt.Errorf("source expects a String path")
	}
	file, err := os.Open(string(path))
	if err != nil {
		return nil, fmt.Errorf("source: %w", err)
	}
	defer file.Close()
	buffered := bufio.NewReader(file)
	const bom = "\xef\xbb\xbf"
	prefix, err := buffered.Peek(len(bom))
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("source header: %w", err)
	}
	if string(prefix) == bom {
		// Peek guarantees these bytes are already buffered.
		_, _ = buffered.Discard(len(bom))
	}
	reader := csv.NewReader(&quotedCRLFReader{source: buffered})
	columns, err := reader.Read()
	if err == io.EOF {
		return runtime.Stream{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("source header: %w", err)
	}
	seen := make(map[string]bool, len(columns))
	for _, name := range columns {
		if name == "" || seen[name] {
			return nil, fmt.Errorf("source: empty or duplicate CSV column %q", name)
		}
		seen[name] = true
	}
	stream := runtime.Stream{Columns: columns}
	for {
		fields, err := reader.Read()
		if err == io.EOF {
			return stream, nil
		}
		if err != nil {
			return nil, fmt.Errorf("source: %w", err)
		}
		record := make(runtime.Record, len(columns))
		for i, field := range fields {
			record[columns[i]] = csvValue(field)
		}
		stream.Values = append(stream.Values, record)
	}
}

// encoding/csv removes the CR in every CRLF sequence, including inside
// quoted fields. Add an extra CR only inside fields so the parser's
// normalization leaves the original bytes intact.
type quotedCRLFReader struct {
	source   *bufio.Reader
	pending  []byte
	readErr  error
	inQuotes bool
}

func (r *quotedCRLFReader) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	if len(r.pending) == 0 {
		if r.readErr != nil {
			err := r.readErr
			r.readErr = nil
			return 0, err
		}
		line, err := r.source.ReadSlice('\n')
		if err == bufio.ErrBufferFull {
			line = bytes.Clone(line)
			var rest []byte
			rest, err = r.source.ReadBytes('\n')
			line = append(line, rest...)
		}
		if len(line) == 0 {
			return 0, err
		}
		r.readErr = err
		// Each escaped quote is a pair and therefore does not change parity.
		if bytes.Count(line, []byte{'"'})%2 != 0 {
			r.inQuotes = !r.inQuotes
		}
		if r.inQuotes && bytes.HasSuffix(line, []byte("\r\n")) {
			// ReadSlice may alias bytes belonging to the next line.
			line = append(bytes.Clone(line[:len(line)-1]), '\r', '\n')
		}
		r.pending = line
	}
	n := copy(dst, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}

func csvValue(field string) runtime.Value {
	if decimal.MatchString(field) {
		if number, err := strconv.ParseFloat(field, 64); err == nil && !math.IsInf(number, 0) {
			return runtime.Number(number)
		}
	}
	return runtime.String(field)
}

func sink(args []runtime.Value) (runtime.Value, error) {
	stream, ok := asStream(args[0])
	if !ok {
		return nil, fmt.Errorf("sink expects a List or Stream")
	}
	path, ok := args[1].(runtime.String)
	if !ok {
		return nil, fmt.Errorf("sink expects a String path")
	}
	if err := writeCSV(string(path), stream); err != nil {
		return nil, fmt.Errorf("sink: %w", err)
	}
	return runtime.Null{}, nil
}

func writeCSV(path string, stream runtime.Stream) error {
	columns := stream.Columns
	if len(columns) == 0 && len(stream.Values) != 0 {
		first, ok := stream.Values[0].(runtime.Record)
		if !ok {
			return fmt.Errorf("CSV row 1 must be a Record")
		}
		for name := range first {
			columns = append(columns, name)
		}
		slices.Sort(columns)
		if len(columns) == 0 {
			return fmt.Errorf("CSV rows must have at least one field")
		}
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".gopherflow-*.csv")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	writer := csv.NewWriter(file)
	if len(columns) != 0 {
		if err := writer.Write(columns); err != nil {
			return err
		}
	}
	fields := make([]string, len(columns))
	for i, value := range stream.Values {
		record, ok := value.(runtime.Record)
		if !ok {
			return fmt.Errorf("CSV row %d must be a Record", i+1)
		}
		if len(record) != len(columns) {
			return fmt.Errorf("CSV row %d does not match the header", i+1)
		}
		for j, name := range columns {
			field, found := record[name]
			if !found {
				return fmt.Errorf("CSV row %d is missing field %q", i+1, name)
			}
			switch field := field.(type) {
			case runtime.String:
				fields[j] = string(field)
			case runtime.Number, runtime.Bool:
				fields[j] = field.Inspect()
			case runtime.Null:
				fields[j] = ""
			default:
				return fmt.Errorf("CSV row %d field %q must be a scalar", i+1, name)
			}
		}
		// A blank CSV line is skipped by the reader. Quote a lone empty cell.
		if len(fields) == 1 && fields[0] == "" {
			writer.Flush()
			if err := writer.Error(); err != nil {
				return err
			}
			if _, err := file.WriteString("\"\"\n"); err != nil {
				return err
			}
		} else if err := writer.Write(fields); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	// Rename replaces the destination inode, so the temporary file must
	// receive the existing regular file's permissions before publication.
	existing, err := os.Lstat(path)
	switch {
	case err == nil:
		if !existing.Mode().IsRegular() {
			return fmt.Errorf("destination %q is not a regular file", path)
		}
		if err := file.Chmod(existing.Mode().Perm()); err != nil {
			return err
		}
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
