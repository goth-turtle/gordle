package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

func run(dict dict, options options) {
	var game state
	game.init(options, dict)

	if options.debug {
		fmt.Printf("[DEBUG] the words are: %v\n\n", game.secrets)
	}

	fmt.Printf("Looking for %d words, with %d letters each.\n"+
		"You have %d guesses, good luck!\n\n",
		options.words, options.chars, options.max_guesses)

	// start game loop
	for game.round = 0; game.round < options.max_guesses; game.round++ {
		game.render(options, dict)

		var this_guess string
		for {
			// prompt for the next guess
			fmt.Print("\n> ")
			n, err := fmt.Scanf("%s", &this_guess)

			if err != nil && strings.Contains(err.Error(), "EOF") {
				game.exit = true
				fmt.Println("")
				break
			} else if err != nil {
				fmt.Printf("error: %v\n", err)
			} else if n != 1 || utf8.RuneCountInString(this_guess) != options.chars {
				fmt.Printf("error: guess needs to be exactly %d "+
					"characters long\n", options.chars)
			} else {
				this_guess = strings.ToUpper(this_guess)

				// ensure the guess is in the dictionary
				i := sort.SearchStrings(dict.words, this_guess)
				if dict.words[i] == this_guess {
					break
				} else {
					fmt.Printf("error: \"%s\" not in dictionary\n", this_guess)
				}
			}
		}

		if game.exit {
			break
		}

		game.guesses[game.round] = this_guess
		game.update(options)

		if game.solved {
			break
		}
	}

	game.render(options, dict)
	if game.solved {
		fmt.Println("\nWell done!")
	} else {
		fmt.Printf("\nThe solutions were: %v\n", game.secrets)
	}
}
