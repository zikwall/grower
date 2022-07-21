package nginx

import (
	"fmt"
	"regexp"
	"strings"
)

type StringParser interface {
	ParseString(line string) (entry *LogEntry, err error)
}

type JSONParser interface {
	ParseJSON(line string) (entry *LogEntry, err error)
}

type Template struct {
	format string
	regexp *regexp.Regexp
}

func NewTemplate(format string) *Template {
	placeholder := " _PLACEHOLDER___ "
	preparedFormat := format
	concatenatedRe := regexp.MustCompile(`[A-Za-z0-9_]\$[A-Za-z0-9_]`)
	for concatenatedRe.MatchString(preparedFormat) {
		preparedFormat = regexp.MustCompile(`([A-Za-z0-9_])\$([A-Za-z0-9_]+)(\\?([^$\\A-Za-z0-9_]))`).ReplaceAllString(
			preparedFormat, fmt.Sprintf("${1}${3}%s$$${2}${3}", placeholder),
		)
	}
	quotedFormat := regexp.QuoteMeta(preparedFormat + " ")
	re := regexp.MustCompile(`\\\$([A-Za-z0-9_]+)(?:\\\$[A-Za-z0-9_])*(\\?([^$\\A-Za-z0-9_]))`).ReplaceAllString(
		quotedFormat, "(?P<$1>[^$3]*)$2")
	re = regexp.MustCompile(fmt.Sprintf(".%s", placeholder)).ReplaceAllString(re, "")
	return &Template{format, regexp.MustCompile(fmt.Sprintf("^%v", strings.Trim(re, " ")))}
}

func (t *Template) ParseString(line string) (entry *LogEntry, err error) {
	re := t.regexp
	fields := re.FindStringSubmatch(line)
	if fields == nil {
		err = fmt.Errorf("access log line '%v' does not match given format '%v'", line, re)
		return
	}
	entry = NewEntry()
	for i, name := range re.SubexpNames() {
		if i == 0 {
			continue
		}
		entry.SetField(name, fields[i])
	}
	return
}

func (t *Template) ParseJSON(_ string) (entry *LogEntry, err error) {
	return nil, err
}
