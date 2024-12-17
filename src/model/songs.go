package model

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/go-msvc/errors"
	"github.com/go-msvc/logger"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	log        = logger.New().WithLevel(logger.LevelDebug)
	titleCaser = cases.Title(language.Afrikaans)
	lowerCaser = cases.Lower(language.Afrikaans)
)

type Song struct {
	SourceFile string   `json:"source_file"`
	Title      string   `json:"title"`
	Artists    []string `json:"artists"`
	Verses     []Verse  `json:"verses"`

	//runtime
	MdFilename string `json:"-"`
}

// if Chorus without any lines, then
type Verse struct {
	Id     int    `json:"id"`
	Chorus bool   `json:"chorus"`
	Bridge bool   `json:"bridge"`
	Lines  []Line `json:"lines"`
}

type Line []Word

type Word struct {
	Text      string `json:"text"`
	KeyChange string `json:"keychange"`
}

func (word Word) MarshalJSON() ([]byte, error) {
	var s string
	if word.KeyChange != "" {
		s = fmt.Sprintf("%s|%s", word.Text, word.KeyChange)
	} else {
		s = word.Text
	}
	return json.Marshal(s)
	// quoted := fmt.Sprintf("\"%s\"", s)
	// log.Debugf("quoted(%s)", quoted)
	// return []byte(quoted), nil
}

func (word *Word) UnmarshalJSON(value []byte) error {
	s := string(value)
	if !strings.HasPrefix(s, "\"") || !strings.HasSuffix(s, "\"") {
		return errors.Errorf("word(%s) is not quoted", s)
	}
	s = s[1 : len(s)-1]
	len, err := fmt.Sscanf(s, "%s|%s", &word.Text, &word.KeyChange)
	if err != nil {
		return errors.Wrapf(err, "failed to parse word(%s) into (text|keyChange)", s)
	}
	if len < 1 {
		return errors.Errorf("no word(%s)", s)
	}
	log.Debugf("Parsed word(%s) -> text(%s),keyChange(%s)", s, word.Text, word.KeyChange)
	return errors.Errorf("NYI")
}

func (song *Song) LoadTxtFile(fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", fn)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lineNr := 0
	expectedLineTypes := lineTypeTitle
	sentenceEnded := true
	var currentVerse *Verse
	for scanner.Scan() {
		lineNr++
		lineText := strings.TrimSpace(scanner.Text())
		// log.Debugf("Line %d: %s", lineNr, lineText)

		//1: title line
		if expectedLineTypes == lineTypeTitle {
			// log.Debugf("Expect title...")
			if lineText == "" {
				return errors.Errorf("empty first line, expected song title")
			}
			song.Title = titleCaser.String(lineText)
			log.Debugf("Title: %s", song.Title)
			expectedLineTypes = lineTypeArtist
			continue
		}

		//optional artists in brackets below the title, allow multiple
		if expectedLineTypes == lineTypeArtist {
			// log.Debugf("Expect artist...")
			if strings.HasPrefix(lineText, "(") && strings.HasSuffix(lineText, ")") {
				song.Artists = append(song.Artists, lineText[1:len(lineText)-1])
				// log.Debugf("%d Artists: %+v", len(song.Artists), song.Artists)
				continue
			}
			//not artist, expect lyrics below
			expectedLineTypes = lineTypeLyrics
		}

		// log.Debugf("Expect lyrics...")
		if len(lineText) == 0 {
			if currentVerse != nil {
				//end current verse
				// log.Debugf("Ending current verse")
				song.Verses = append(song.Verses, *currentVerse)
				currentVerse = nil
			}
			// log.Debugf("Skip empty line")
			continue
		}

		//start a verse or append to current verse
		if currentVerse == nil {
			currentVerse = &Verse{
				Id:     len(song.Verses),
				Chorus: false,
				Lines:  []Line{},
			}
		}
		//if first line of verse indicates chorus
		if lineText == "Koor:" {
			currentVerse.Chorus = true
			continue
		}

		if lineText == "Brug:" {
			currentVerse.Bridge = true
			continue
		}

		// newLine := Line{}
		//if line is mostly uppercase, convert to sentence
		r := countCharSets(lineText)
		if len(r.unknown) > 0 {
			return errors.Errorf("invalid lyrics chars(%s) in %s(%d): %s", r.unknownChars(), fn, lineNr, lineText)
		}
		if r.upper > r.lower {
			lineText = lowerCaser.String(lineText)
			log.Debugf("Lower cased %s(%d): %s", fn, lineNr, lineText)

			//if last line ended a sentence, make first word uppercase
			if sentenceEnded && unicode.IsLower(rune(lineText[0])) {
				lineText = string(unicode.ToUpper(rune(lineText[0]))) + lineText[1:]
			}

			sentenceEnded = strings.Contains(endOfSentenceChars, string(lineText[len(lineText)-1]))
		}

		words := strings.Split(lineText, " ")
		newLine := Line([]Word{})
		for _, word := range words {
			word = strings.TrimSpace(word)
			if word == "" {
				continue
			}

			//log.Debugf("word(%s)", word)
			w := Word{Text: word}
			newLine = append(newLine, w)
		}
		currentVerse.Lines = append(currentVerse.Lines, newLine)
	}

	//complete the last verse (if had no empty line after)
	if currentVerse != nil {
		//log.Debugf("Ending last verse")
		song.Verses = append(song.Verses, *currentVerse)
		currentVerse = nil
	}
	return nil
} //Song.LoadTxtFile()

type lineType int

const (
	lineTypeTitle lineType = iota
	lineTypeArtist
	lineTypeEmpty
	lineTypeLyrics
)

func (song Song) ExportMarkDown(fn string) error {
	f, err := os.Create(fn)
	if err != nil {
		return errors.Wrapf(err, "failed to create mark down file %s", fn)
	}
	defer f.Close()

	//title
	fmt.Fprintf(f, "# %s\n", song.Title)

	//artists (optional)
	if len(song.Artists) > 0 {
		for i, a := range song.Artists {
			if i == 0 {
				fmt.Fprintf(f, "## (")
			} else {
				fmt.Fprintf(f, ", ")
			}
			fmt.Fprint(f, a)
		}
		fmt.Fprint(f, ")\n")
	}

	//verses
	for _, verse := range song.Verses {
		fmt.Fprint(f, "\n") //blank line to separate from previous verse

		//decorations
		if verse.Chorus {
			fmt.Fprint(f, "_Koor:_\n\n")
		}
		if verse.Bridge {
			fmt.Fprint(f, "_Brug:_\n\n")
		}
		for _, line := range verse.Lines {
			s := ""
			for i, word := range line {
				if i > 0 {
					s += " "
				}
				s += word.Text
			}
			fmt.Fprintf(f, "%s\n\n", s)
		}
	}
	return nil
} //ExportMarkDown

type charSetCount struct {
	lower   int
	upper   int
	symbol  int
	space   int
	unknown map[rune]bool
}

func (c charSetCount) unknownChars() string {
	s := ""
	for ch := range c.unknown {
		s += string(ch)
	}
	return s
} //charSetCount.unknownChars()

const (
	allowedTextSymbols = `!?;:,.-'"()[]0123456789/`
	endOfSentenceChars = `.!?;`
)

func countCharSets(s string) charSetCount {
	result := charSetCount{
		unknown: map[rune]bool{},
	}
	for _, c := range s {
		switch {
		case c == ' ':
			result.space++
		case unicode.IsUpper(c):
			result.upper++
		case unicode.IsLower(c):
			result.lower++
		case strings.Contains(allowedTextSymbols, string(c)):
			result.symbol++
		default:
			result.unknown[c] = true
		}
	}
	return result
} //countChars
