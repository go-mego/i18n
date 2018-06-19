package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"

	"html/template"

	"github.com/go-mego/mego"
	yaml "gopkg.in/yaml.v2"
)

var (
	ErrNoFallback = errors.New("i18n: no fallback language file")
)

func New(option *Options) mego.HandlerFunc {
	files, err := ioutil.ReadDir(option.Directory)
	if err != nil {
		panic(err)
	}

	var translations []*translation
	translationMap := make(map[string]*translation)

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		//
		content, err := ioutil.ReadFile(option.Directory + "/" + f.Name())
		if err != nil {
			panic(err)
		}

		//
		language := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
		rawTranslation := make(map[string]interface{})

		//
		switch filepath.Ext(f.Name()) {
		case ".json":
			err := json.Unmarshal(content, &rawTranslation)
			if err != nil {
				panic(err)
			}
		case ".yml":
			err := yaml.Unmarshal(content, &rawTranslation)
			if err != nil {
				panic(err)
			}
		default:
			continue
		}

		//
		rawTranslation = flatten(rawTranslation)
		//
		cookedTransation := make(map[string]string)
		for k, v := range rawTranslation {
			cookedTransation[k] = v.(string)
		}

		t := &translation{
			language: language,
			strings:  cookedTransation,
		}

		//
		split := strings.Split(language, "-")
		translationMap[strings.ToUpper(split[0])] = t
		translationMap[strings.ToUpper(language)] = t

		//
		translations = append(translations, t)

		//debug ::Loaded language xx

	}

	if _, ok := translationMap[strings.ToUpper(option.FallbackLanguage)]; !ok {
		panic(ErrNoFallback)
	}

	return func(c *mego.Context) {
		l := &Locale{
			context:        c,
			translationMap: translationMap,
			translations:   translations,
		}

		acceptLanguage := getLanguagePriority(c.Request.Header.Get("Accept-Language"))

		for _, v := range acceptLanguage {
			if l.hasTranslation(strings.ToUpper(v)) {
				l.SetLanguage(strings.ToUpper(v))
				break
			}
		}
		if l.usingTranslation == nil {
			l.SetLanguage(strings.ToUpper(l.options.FallbackLanguage))
		}

		c.Map(l)
		c.Next()
	}
}

//
//
// Chrome: [zh-TW,zh;q=0.8,en-US;q=0.6,en;q=0.4]
//		Safari: [zh-tw]
//		FireFox: [zh-TW,zh;q=0.8,en-US;q=0.5,en;q=0.3]
func getLanguagePriority(header string) (priority []string) {
	for _, v := range strings.Split(header, ",") {
		info := strings.Split(v, ";")
		if len(info) > 1 {
			priority = append(priority, info[0])
			continue
		}
		priority = append(priority, v)
	}
	return
}

func (l *Locale) hasTranslation(language string) bool {
	_, ok := l.translationMap[language]
	return ok
}

func (l *Locale) getTranslation(language string) *translation {
	if v, ok := l.translationMap[language]; ok {
		return v
	}
	return l.translationMap[l.options.FallbackLanguage]
}

type Options struct {
	Directory        string
	FallbackLanguage string
	//
	Parameter string
}

type Locale struct {
	context          *mego.Context
	options          *Options
	translationMap   map[string]*translation
	translations     []*translation
	usingTranslation *translation
}

type translation struct {
	language string
	strings  map[string]string
}

func (t *translation) find(key string) string {
	v, ok := t.strings[key]
	if !ok {
		return ""
	}
	return v
}

func (t *translation) apply(trans string, arg ...interface{}) string {
	switch len(arg) {
	case 0:
		return trans
	case 1:
		switch v := arg[0].(type) {
		case map[string]string:
			for k, v := range v {
				trans = strings.Replace(trans, fmt.Sprintf("{%s}", k), v, -1)
			}
			return trans
		default:
			return trans
		}
	default:
		return fmt.Sprintf(trans, arg...)
	}
}

func (l *Locale) TemplateFunc() template.FuncMap {
	return map[string]interface{}{
		"_": l.Get,
	}
}

func (l *Locale) Get(key string, args ...interface{}) string {
	trans := l.usingTranslation.find(key)
	return l.usingTranslation.apply(trans, args...)
}

func (l *Locale) Language() string {
	return ""
}

func (l *Locale) SetLanguage(lang string) {
	l.usingTranslation = l.getTranslation(lang)
}

// Source: https://github.com/doublerebel/bellows/blob/master/main.go
func flatten(value interface{}) map[string]interface{} {
	return flattenPrefixed(value, "")
}

func flattenPrefixed(value interface{}, prefix string) map[string]interface{} {
	m := make(map[string]interface{})
	flattenPrefixedToResult(value, prefix, m)
	return m
}

func flattenPrefixedToResult(value interface{}, prefix string, m map[string]interface{}) {
	base := ""
	if prefix != "" {
		base = prefix + "."
	}

	original := reflect.ValueOf(value)
	kind := original.Kind()
	if kind == reflect.Ptr || kind == reflect.Interface {
		original = reflect.Indirect(original)
		kind = original.Kind()
	}
	t := original.Type()

	switch kind {
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			break
		}
		for _, childKey := range original.MapKeys() {
			childValue := original.MapIndex(childKey)
			flattenPrefixedToResult(childValue.Interface(), base+childKey.String(), m)
		}
	case reflect.Struct:
		for i := 0; i < original.NumField(); i += 1 {
			childValue := original.Field(i)
			childKey := t.Field(i).Name
			flattenPrefixedToResult(childValue.Interface(), base+childKey, m)
		}
	default:
		if prefix != "" {
			m[prefix] = value
		}
	}
}
