package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

const program_name string = "gordle"
const version_major int = 0
const version_minor int = 1
const version_patch int = 0
const default_dict_path string = "/usr/share/dict:/usr/dict"
const default_dict string = "words"
const help_msg string = program_name + " v%d.%d.%d" +
	"\nUSAGE:" +
	"\n    %s [FLAGS] [OPTIONS]" +
	"\n" +
	"\nFLAGS:" +
	"\n    -h, --help         Show this help message and exit" +
	"\n    -v, --version      Show program name and version and exit" +
	"\n        --debug        Show debug messages, including solutions" +
	"\n" +
	"\nOPTIONS:" +
	"\n    -l, --list         Name of the dictionary file" +
	"\n                           [default: \"" + default_dict + "\"]" +
	"\n    -d, --dicts        Colon-separated list of directories in which" +
	"\n                       to search for dictionary files" +
	"\n                           [default: \"" + default_dict_path + "\"]" +
	"\n    -w, --words        Number of words to guess" +
	"\n                           [default: 1]" +
	"\n    -g, --guesses      Maximum number of guesses" +
	"\n                           [default: words + 5]" +
	"\n    -c, --chars        The length (in utf-8 code points) of words" +
	"\n                           [default: 5]" +
	"\n    -a, --force-ascii  Only accept dictionary words that" +
	"\n                       consist exclusively of letters in a-z and A-Z" +
	"\n                           [default: true]" +
	"\n"

type options struct {
	dict_path   string
	language    string
	force_ascii bool
	debug       bool
	max_guesses int
	words       int
	chars       int
}

func main() {
	options, err := parse_args(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n\n", err)
		fmt.Fprintf(os.Stderr, help_msg, version_major, version_minor,
			version_patch, os.Args[0])
		os.Exit(1)
	}

	file, err := find_dict(options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
		os.Exit(1)
	}
	defer file.Close()

	run(parse_dict(options, file), options)
}

func parse_args(args []string) (options options, err error) {
	options.language = default_dict
	options.dict_path = default_dict_path
	options.force_ascii = true
	options.debug = false
	options.chars = 5
	options.words = 1

	n, err := parse_flags(args, &options)
	if err == nil {
		err = parse_options(args[n:], &options)
	}

	return
}

func parse_flags(args []string, options *options) (consumed int, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if len(arg) < 2 || arg[0] != '-' {
			return 0, fmt.Errorf("error at \"%s\": expected flag or option", arg)
		}

		var flag string
		if arg[1] == '-' { // long flag
			flag = arg[2:]
		} else {
			flag = arg[1:]
		}

		switch flag {
		case "h", "help":
			fmt.Printf(help_msg, version_major, version_minor, version_patch,
				os.Args[0])
			os.Exit(0)
		case "v", "version":
			fmt.Printf("%s v%d.%d.%d\n", program_name, version_major,
				version_minor, version_patch)
			os.Exit(0)
		case "debug":
			options.debug = true
		default:
			return i, nil
		}
	}

	return len(args), nil
}

func parse_options(args []string, options *options) (err error) {
	set_guesses := false

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// make sure the current argument could be a valid option
		if len(arg) < 2 || arg[0] != '-' {
			return fmt.Errorf(
				"error at \"%s\": expected option", arg)
		}

		var opt string
		var value string

		if arg[1] != '-' && len(arg) > 2 { // short option, value in same token
			opt = arg[1:2]
			value = arg[2:]
		} else { // value is in the next token
			if i+1 >= len(args) {
				return fmt.Errorf(
					"error at \"%s\": missing value to option", arg)
			}

			if arg[1] == '-' { // long option
				opt = arg[2:]
				value = args[i+1]
			} else { // short option
				opt = arg[1:]
				value = args[i+1]
			}
			i++
		}

		switch opt {
		case "l", "list":
			options.language = value
		case "d", "dicts":
			options.dict_path = value
		case "w", "words":
			var n uint64
			n, err = strconv.ParseUint(value, 10, 32)
			options.words = int(n)
		case "g", "guesses":
			var n uint64
			n, err = strconv.ParseUint(value, 10, 32)
			options.max_guesses = int(n)
			set_guesses = true
		case "c", "chars":
			var n uint64
			n, err = strconv.ParseUint(value, 10, 32)
			options.chars = int(n)
		case "a", "force-ascii":
			switch value {
			case "y", "yes", "true":
				options.force_ascii = true
			case "n", "no", "false":
				options.force_ascii = false
			default:
				err = errors.New("unrecognized value, use one of: " +
					"y, yes, true, n, no, false")
			}
		default:
			err = errors.New("unrecognized option")
		}

		if err != nil {
			err = fmt.Errorf("error at \"%s\": %v", arg, err)
			return
		}
	}

	if !set_guesses {
		options.max_guesses = 3 + options.words + options.chars/2
	}

	return nil
}
