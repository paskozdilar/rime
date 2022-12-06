package rime

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Rime interface {
	Close() error
	Channel() chan string
}

type rime struct {
	word      string
	syllables int
	streamer  chan string
	closer    chan struct{}
}

func (r rime) Close() (err error) {
	defer func() {
		if msg := recover(); msg != nil {
			err = errors.New(fmt.Sprintf("%s", msg))
		}
	}()
	close(r.closer)
	return
}

func (r rime) Channel() chan string {
	return r.streamer
}

func (r rime) worker() {
	var (
		exclude []string
		words   []string
		more    bool = true
		err     error
	)

	defer close(r.streamer)

	for more {
		words, more, err = GetRhymesExclude(r.word, r.syllables, exclude)
		if err != nil {
			log.Fatalln("GetRhymes:", err)
		}
		for _, word := range words {
			select {
			case r.streamer <- word:
				exclude = append(exclude, word)
			case <-r.closer:
				return
			}
		}
	}
}

func NewRime(word string, syllables int) Rime {
	r := rime{
		word:      word,
		syllables: syllables,
		streamer:  make(chan string),
		closer:    make(chan struct{}),
	}
	go r.worker()
	return r
}

type Word struct {
	Text, Note string
}

type WordList struct {
	Words []Word
	More  bool
}

func GetRhymes(word string, syllables int) (words []string, more bool, err error) {
	return GetRhymesExclude(word, syllables, nil)
}

func GetRhymesExclude(word string, syllables int, exclude []string) (words []string, more bool, err error) {
	if syllables <= 0 {
		err = errors.New("syllables cannot be non-positive")
		return
	}

	var excludeStr string
	if exclude == nil {
		excludeStr = word
	} else {
		for i, str := range exclude {
			if i == 0 {
				excludeStr = str
			} else {
				excludeStr += "," + str
			}
		}
	}

	v := url.Values{}
	v.Set("search_type", "rhyme")
	v.Set("search_subtype", "vowel")
	v.Set("ending", word)
	v.Set("pronunciation", word)
	v.Set("n_syllables", strconv.Itoa(syllables))
	v.Set("exclude", excludeStr)
	body := v.Encode()

	resp, err := http.Post(
		"https://rime.com.hr/more_results",
		"application/x-www-form-urlencoded",
		strings.NewReader(body),
	)
	if err != nil {
		err = errors.New(fmt.Sprintf("http.Post: %s", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = errors.New(fmt.Sprintf("http.Post: %s", resp.Status))
		return
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		err = errors.New(fmt.Sprintf("io.ReadAll: %s", resp.Status))
		return
	}

	var wordList WordList
	err = json.Unmarshal(buf, &wordList)
	if err != nil {
		err = errors.New(fmt.Sprintf("json.Unmarshal: %s", err))
		return
	}

	for _, word := range wordList.Words {
		words = append(words, word.Text)
	}
	more = wordList.More
	return
}
