package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
)

type dict struct {
	words      []string
	letter_set []rune
}

func find_dict(options options) (file *os.File, err error) {
	for _, dir := range strings.Split(options.dict_path, ":") {
		file, err = os.Open(dir + "/" + options.language)
		if err == nil {
			return
		}
	}

	return nil, fmt.Errorf("could not find language \"%s\"", options.language)
}

func parse_dict(options options, file *os.File) (dict dict) {
	scanner := bufio.NewScanner(file)
	letters := make(map[rune]bool)

	for scanner.Scan() {
		line := strings.ToUpper(scanner.Text())

		// pick words of the right length
		if utf8.RuneCountInString(line) == options.chars {
			// reject words with diacritics and similar
			// if force-ascii is enabled
			use_word := true
			if options.force_ascii {
				for _, char := range line {
					if char < 'A' || char > 'Z' {
						use_word = false
						break
					}
				}
			}

			// if the word is used, then append it to the dict,
			// and update the used characters list
			if use_word {
				dict.words = append(dict.words, line)

				for _, char := range line {
					letters[char] = true
				}
			}
		}
	}

	// convert map of letters to a slice
	dict.letter_set = make([]rune, len(letters))
	i := 0
	for key := range letters {
		dict.letter_set[i] = key
		i++
	}

	sort.Strings(dict.words)
	sort.Slice(dict.letter_set,
		func(i, j int) bool { return dict.letter_set[i] < dict.letter_set[j] })

	return
}
