package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-msvc/errors"
	"github.com/go-msvc/logger"
	"github.com/jansemmelink/singsaam/src/model"
)

var log = logger.New().WithLevel(logger.LevelDebug)

func main() {
	inputDir := flag.String("input", "../../files", "Input directory")
	dataDir := flag.String("data", "../../data", "Data directory")

	findNew := flag.Int("find-new", 0, "Find files in input that are not yet in data")
	export := flag.Bool("export-json", false, "Exports JSON files into data/song_#.json")
	flag.Parse()

	tool := tool{
		inputDir: *inputDir,
		dataDir:  *dataDir,
		songs:    []model.Song{},
	}

	switch {
	case *findNew > 0:
		if err := tool.findNew(*findNew); err != nil {
			log.Errorf("failed: %+v ... but continue to save", err)
		}
	default:
		panic("no action specified")
	}

	tool.checkArtists()
	log.Debugf("Tool has %d songs", len(tool.songs))

	if *export {
		for i, s := range tool.songs {
			fn := fmt.Sprintf("%s/song_%d.json", *dataDir, i)
			f, err := os.Create(fn)
			if err != nil {
				panic(fmt.Sprintf("failed to create song file %s: %+v", fn, err))
			}
			defer f.Close()
			e := json.NewEncoder(f)
			e.SetIndent("", "  ")
			if err := e.Encode(s); err != nil {
				panic(fmt.Sprintf("failed to encode song(%s): %+v", s.Title, err))
			}
		}
		log.Debugf("Written to %s", *dataDir)
	}
}

type tool struct {
	inputDir string
	dataDir  string

	songs []model.Song
}

func (tool *tool) findNew(limit int) error {
	if err := filepath.Walk(tool.inputDir, tool.evalInputFile(limit)); err != nil {
		return errors.Wrapf(err, "failed to process all files in %s", tool.inputDir)
	}
	return nil
} //tool.findNew()

func (tool *tool) evalInputFile(limit int) func(path string, info fs.FileInfo, err error) error {
	return func(path string, info fs.FileInfo, err error) error {
		if info.Mode().IsRegular() {
			if len(tool.songs) > limit {
				return errors.Errorf("Stop after 5")
			}
			//process regular file, expexting it to be .txt with song lirics
			// log.Debugf("file: %s", path)
			if !strings.HasSuffix(path, ".txt") {
				return errors.Errorf("%s is not a .txt file", path)
			}

			newSong := model.Song{
				Title:      "?",
				SourceFile: path,
			}
			if err := newSong.LoadTxtFile(path); err != nil {
				return errors.Wrapf(err, "failed to read song file %s", path)
			}
			tool.songs = append(tool.songs, newSong)
		}
		return nil
	}
}

func (tool tool) checkArtists() {
	artists := map[string]bool{}
	for _, s := range tool.songs {
		for _, artist := range s.Artists {
			artists[artist] = true
		}
	}

	list := []string{}
	for artist := range artists {
		list = append(list, artist)
	}
	sort.Slice(list, func(i, j int) bool { return list[i] > list[j] })
	for _, artist := range list {
		log.Debugf("%s", artist)
	}
}
