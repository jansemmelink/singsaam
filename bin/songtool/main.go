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
	exportJson := flag.String("export-json", "../../generated/json", "Exports JSON files into song_#.json")
	exportMd := flag.String("export-md", "../../generated/md", "Exports Markdown files into song_#.md")
	flag.Parse()

	tool := tool{
		inputDir: *inputDir,
		dataDir:  *dataDir,
		songs:    []*model.Song{},
	}

	switch {
	case *findNew > 0:
		if err := tool.findNew(*findNew); err != nil {
			panic(fmt.Sprintf("failed: %+v", err))
		}
	default:
		panic("no action specified")
	}

	//checks
	tool.checkArtists()
	if err := tool.checkText(); err != nil {
		panic(fmt.Sprintf("failed: %+v", err))
	}
	//tool.checkChorus()

	log.Debugf("Tool has %d songs", len(tool.songs))

	if *exportJson != "" {
		if err := tool.exportJson(*exportJson); err != nil {
			panic(fmt.Sprintf("failed: %+v", err))
		}
	} //export json

	if *exportMd != "" {
		if err := tool.exportMd(*exportMd); err != nil {
			panic(fmt.Sprintf("failed: %+v", err))
		}
	} //export md
}

type tool struct {
	inputDir string
	dataDir  string

	songs []*model.Song
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
			tool.songs = append(tool.songs, &newSong)
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
} //tool.checkArtists()

func (tool tool) checkText() error {
	// for _, s := range tool.songs {
	// 	r := countCharSets(s.Title)
	// 	if r.upper >= r.lower {
	// 		return errors.Errorf("Excessive uppercase in title: %s", s.Title)
	// 	}
	// 	if len(r.unknown) > 0 {
	// 		return errors.Errorf("Unknown text characters(%s) in title %s", r.unknownChars(), s.Title)
	// 	}
	// }
	return nil
} //tool.checkCase()

func (tool tool) exportJson(dir string) error {
	for i, s := range tool.songs {
		fn := fmt.Sprintf("%s/song_%d.json", dir, i)
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
	log.Debugf("Written Json to %s", dir)
	return nil
} //tool.exportJson()

func (tool tool) exportMd(dir string) error {
	titlesFilename := fmt.Sprintf("%s/_titles.md", dir)
	titlesFile, err := os.Create(titlesFilename)
	if err != nil {
		return errors.Wrapf(err, "failed to create md index file %s", titlesFilename)
	}
	defer titlesFile.Close()

	fmt.Fprintf(titlesFile, "# Titles\n")
	sort.Slice(tool.songs, func(i, j int) bool { return tool.songs[i].Title < tool.songs[j].Title })
	byArtist := map[string][]*model.Song{}
	for i, s := range tool.songs {
		s.MdFilename = fmt.Sprintf("%s/song_%d.md", dir, i)
		if err := s.ExportMarkDown(s.MdFilename); err != nil {
			return errors.Wrapf(err, "failed")
		}
		fmt.Fprintf(titlesFile, "* [%s](%s)\n", s.Title, s.MdFilename)
		for _, artist := range s.Artists {
			list, ok := byArtist[artist]
			if !ok {
				byArtist[artist] = []*model.Song{s}
			} else {
				byArtist[artist] = append(list, s)
			}
		}
	} //for each song

	artists := []string{}
	for artist := range byArtist {
		artists = append(artists, artist)
	}
	sort.Slice(artists, func(i, j int) bool { return artists[i] < artists[j] })

	artistsFilename := fmt.Sprintf("%s/_artists.md", dir)
	artistsFile, err := os.Create(artistsFilename)
	if err != nil {
		return errors.Wrapf(err, "failed to create md index file %s", artistsFilename)
	}
	defer artistsFile.Close()
	fmt.Fprintf(artistsFile, "# Artists\n")

	for _, artist := range artists {
		list := byArtist[artist]
		fmt.Fprintf(artistsFile, "## %s\n", artist)
		for _, song := range list {
			fmt.Fprintf(artistsFile, "* [%s](%s)\n", song.Title, song.MdFilename)
		}
	}

	log.Debugf("Written MD to %s", dir)
	return nil
} //tool.exportMd()
