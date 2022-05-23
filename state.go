package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type state struct {
	secrets []string
	guesses []string
	columns []column
	round   int
	solved  bool
	exit    bool

	// precomputed strings and values

	empty_word      string
	empty_letters   string
	horizontal_line string
	padding         int

	// separators for the guess rows, with padding spaces
	separator_left    string
	separator_right   string
	separator_between string

	// horizontal padding space for the letter display in the lower area
	space_left    string
	space_between string
}

type column struct {
	hints  []string
	solved bool

	wrong_letters     map[rune]bool
	contained_letters map[rune]int // includes perfect letters in its count
	perfect_letters   map[rune]bitfield
}

type display_builder struct {
	count       int
	wrap        int
	lines       []string
	active_line *strings.Builder
}

type bitfield uint32

// regular letters
const regular_format string = "%c"

// wrong letters
const wrong_format string = "\x1B[41m\x1B[30m%c\x1B[0m"

// black on yellow letters
const misplaced_format string = "\x1B[43m\x1B[30m%c\x1B[0m"

// black on green letters
const perfect_format string = "\x1B[42m\x1B[30m%c\x1B[0m"

func (s *state) init(options options, dict dict) {
	s.round = 0
	s.exit = false
	s.solved = false
	s.secrets = make([]string, options.words)
	s.guesses = make([]string, options.max_guesses)
	s.columns = make([]column, options.words)

	s.padding = 6 - options.chars/2
	if s.padding < 1 {
		s.padding = 1
	}

	s.empty_word = times('.', options.chars)
	s.empty_letters = times(' ', options.chars+2*s.padding)
	s.horizontal_line = times('-',
		options.words*(options.chars+2*s.padding)+options.words+1)
	s.separator_left = "|" + times(' ', s.padding)
	s.separator_right = times(' ', s.padding) + "|"
	s.separator_between = times(' ', s.padding) + "|" + times(' ', s.padding)
	s.space_between = " "
	s.space_left = " "

	// use current system time as rng seed
	rand.Seed(time.Now().Unix())

	for i := 0; i < options.words; i++ {
		s.secrets[i] = dict.words[rand.Intn(len(dict.words))]
	}

	for i := 0; i < options.words; i++ {
		s.columns[i].perfect_letters = make(map[rune]bitfield)
		s.columns[i].contained_letters = make(map[rune]int)
		s.columns[i].wrong_letters = make(map[rune]bool)

		s.columns[i].hints = make([]string, options.max_guesses)
		s.columns[i].solved = false
		for r := 0; r < options.max_guesses; r++ {
			s.columns[i].hints[r] = s.empty_word
		}
	}
}

func (s *state) update(options options) {
	guess := s.guesses[s.round]
	s.solved = true

	for w := 0; w < options.words; w++ {
		col := &(s.columns[w])

		// generate new hints
		var hint string
		var perfect, misplaced []bool
		if col.solved {
			hint = s.empty_word
		} else {
			var solved bool
			hint, solved, perfect, misplaced =
				generate_hint(options, guess, s.secrets[w])
			col.solved = col.solved || solved
		}

		// update list
		col.hints[s.round] = hint
		s.solved = s.solved && col.solved

		// update letters
		if !col.solved {
			temp_contained := make(map[rune]int)
			for i, letter := range guess {
				if perfect[i] {
					col.perfect_letters[letter] = col.perfect_letters[letter].
						set(uint(i), true)
					change_count(temp_contained, letter, 1, 1)
					delete(col.wrong_letters, letter)
				} else if misplaced[i] {
					change_count(temp_contained, letter, 1, 1)
					delete(col.wrong_letters, letter)
				} else if col.perfect_letters[letter] == 0 &&
					col.contained_letters[letter] == 0 &&
					temp_contained[letter] == 0 {

					col.wrong_letters[letter] = true
				}
			}
			// for contained letters, make sure to use
			// the maximum between the previous count and the new one
			for _, letter := range guess {
				if temp_contained[letter] > col.contained_letters[letter] {
					col.contained_letters[letter] = temp_contained[letter]
				}
			}
		}
	}
}

func (s *state) render(options options, dict dict) {
	output := new(strings.Builder)

	// horizontal line on top of the table
	fmt.Fprintln(output, s.horizontal_line)

	// hint rows
	cols := len(s.columns)
	for r := 0; r < options.max_guesses; r++ {
		// vertical line on the left side
		fmt.Fprint(output, s.separator_left)

		// vertical lines between columns
		for c := 0; c+1 < cols; c++ {
			fmt.Fprintf(output, "%s%s", s.columns[c].hints[r],
				s.separator_between)
		}

		// vertical line on the right side
		fmt.Fprintf(output, "%s%s\n", s.columns[cols-1].hints[r],
			s.separator_right)
	}

	// horizontal line at the bottom of the table
	fmt.Fprintf(output, "%s\n\n", s.horizontal_line)

	// build letter display of each column
	builders := make([]*display_builder, options.words)
	displays := make([][]string, options.words)
	max_rows := 0
	for c, col := range s.columns {
		builders[c] = new(display_builder)
		builders[c].init(options.chars + 2*s.padding)

		// append each character the appropriate number of times
		// and in the right color to the builder
		for _, char := range dict.letter_set {
			printed := false
			amount_perfect := 0

			if contains_bitfield(col.perfect_letters, char) {
				amount_perfect = col.perfect_letters[char].ones()
				builders[c].append_times(fmt.Sprintf(perfect_format, char),
					amount_perfect)
				printed = true
			}
			if contains_int(col.contained_letters, char) {
				builders[c].append_times(fmt.Sprintf(misplaced_format, char),
					col.contained_letters[char]-amount_perfect)
				printed = true
			}
			if contains_bool(col.wrong_letters, char) {
				builders[c].append(fmt.Sprintf(wrong_format, char))
				printed = true
			}
			if !printed {
				builders[c].append(fmt.Sprintf(regular_format, char))
			}
		}

		displays[c] = builders[c].finish()
		if !col.solved && len(displays[c]) > max_rows {
			max_rows = len(displays[c])
		}
	}

	// now print the letter displays
	for r := 0; r < max_rows; r++ {
		output.WriteString(s.space_left)

		for c, disp := range displays {
			if !s.columns[c].solved && r < len(disp) {
				output.WriteString(disp[r])
			} else {
				output.WriteString(s.empty_letters)
			}
			output.WriteString(s.space_between)
		}

		output.WriteString("\n")
	}

	fmt.Print(output)
}

func (db *display_builder) init(wrap int) {
	db.active_line = new(strings.Builder)
	db.wrap = wrap
}

func (db *display_builder) append(str string) {
	db.active_line.WriteString(str)
	db.count++

	if db.count >= db.wrap {
		db.lines = append(db.lines, db.active_line.String())
		db.active_line.Reset()
		db.count = 0
	}
}

func (db *display_builder) append_times(str string, amount int) {
	for i := 0; i < amount; i++ {
		db.append(str)
	}
}

func (db *display_builder) finish() (lines []string) {
	if db.count != 0 {
		db.active_line.WriteString(times(' ', db.wrap-db.count))
		db.lines = append(db.lines, db.active_line.String())
		db.count = 0
	}

	return db.lines
}

func (b bitfield) set(index uint, bit bool) (result bitfield) {
	if bit {
		result = b | (1 << index)
	} else {
		result = b &^ (1 << index)
	}

	return
}

func (b bitfield) get(index uint) (bit bool) {
	return b&(1<<index) != 0
}

func (b bitfield) ones() (amount int) {
	for amount = 0; b != 0; amount++ {
		// clear the least significant bit each time until b is 0
		b &= b - 1
	}

	return
}

func change_count(list map[rune]int, letter rune, change int, init int) {
	count, exists := list[letter]

	if exists && count+change > 0 { // update count unless it would get 0
		list[letter] = count + change
	} else if exists { // if the count would drop to 0, delete the letter
		delete(list, letter)
	} else if init != 0 { // initialize the letter unless init is 0
		list[letter] = init
	}
}

func contains_bool(list map[rune]bool, char rune) (contains bool) {
	_, contains = list[char]
	return
}

func contains_int(list map[rune]int, char rune) (contains bool) {
	_, contains = list[char]
	return
}

func contains_bitfield(list map[rune]bitfield, char rune) (contains bool) {
	_, contains = list[char]
	return
}

func times(char rune, amount int) (result string) {
	builder := new(strings.Builder)
	for i := 0; i < amount; i++ {
		fmt.Fprintf(builder, "%c", char)
	}
	return builder.String()
}

func generate_hint(options options, guess string, secret string) (hint string,
	solved bool, perfect_letters []bool, misplaced_letters []bool) {

	hint_builder := new(strings.Builder)
	runes_secret := []rune(secret)
	char_counts := make(map[rune]int)
	perfect_letters = make([]bool, options.chars)
	misplaced_letters = make([]bool, options.chars)
	solved = true

	// count secret characters
	for _, char := range runes_secret {
		n, exists := char_counts[char]
		if exists {
			char_counts[char] = n + 1
		} else {
			char_counts[char] = 1
		}
	}

	// find perfect matches
	for i, char := range guess {
		if char == runes_secret[i] {
			perfect_letters[i] = true
			char_counts[char] -= 1
		} else {
			solved = false
		}
	}

	// find misplaced matches
	for i, char := range guess {
		n, exists := char_counts[char]
		if exists && n > 0 {
			misplaced_letters[i] = true
			char_counts[char] = n - 1
		}
	}

	// append formatted letters
	for i, char := range guess {
		if perfect_letters[i] {
			fmt.Fprintf(hint_builder, perfect_format, char)
		} else if misplaced_letters[i] {
			fmt.Fprintf(hint_builder, misplaced_format, char)
		} else {
			fmt.Fprintf(hint_builder, regular_format, char)
		}
	}

	hint = hint_builder.String()
	return
}
