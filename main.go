package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time" // Added for loading effects

	"github.com/aandrew-me/tgpt/v2/src/bubbletea"
	"github.com/aandrew-me/tgpt/v2/src/helper"
	"github.com/aandrew-me/tgpt/v2/src/imagegen"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/utils"
	Prompt "github.com/c-bata/go-prompt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
)

const localVersion = "2.10.0"

// Define a richer color palette
var (
	bold      = color.New(color.Bold)
	blue      = color.New(color.FgBlue)
	green     = color.New(color.FgGreen)
	yellow    = color.New(color.FgYellow)
	cyan      = color.New(color.FgCyan)
	magenta   = color.New(color.FgMagenta)
	red       = color.New(color.FgRed) // For general red text, not necessarily errors
	white     = color.New(color.FgWhite)
	boldWhite = color.New(color.FgWhite, color.Bold)
	boldGreen = color.New(color.FgGreen, color.Bold)
	boldYellow= color.New(color.FgYellow, color.Bold)
)

var programLoop = true

// Function to simulate a more advanced loading effect
func showLoading(message string) {
	spinnerChars := []string{"⢿", "⣻", "⣽", "⣾", "⣷", "⣯", "⣟", "⡿"} // Braille spinner
	loadingColors := []*color.Color{
		blue,
		color.New(color.FgHiBlue),
		cyan,
		color.New(color.FgHiCyan),
		green,
		color.New(color.FgHiGreen),
		magenta,
		color.New(color.FgHiMagenta),
		yellow,
		color.New(color.FgHiYellow),
	}
	messageColor := white // Color for the loading message text
	cycles := 3           // Number of times to cycle through the spinnerChars for a decent duration
	interval := 120 * time.Millisecond

	messageColor.Print(message + " ") // Print initial message part

	for i := 0; i < len(spinnerChars)*cycles; i++ {
		currentColor := loadingColors[i%len(loadingColors)]
		// \r to return to the beginning of the line
		fmt.Printf("\r")                    // Go to beginning of line
		messageColor.Print(message + " ")   // Reprint message with its color
		currentColor.Print(spinnerChars[i%len(spinnerChars)]) // Print colored spinner
		time.Sleep(interval)
	}
	// Clear the spinner and loading message part, then print "Done!"
	clearLength := 0
	// Calculate actual display length of message and spinner (ansi codes don't count)
	// For simplicity, using len(message) + 1 for spinner + 1 for space.
	// This might need adjustment if message contains many multi-byte chars.
	clearLength = len(message) + 2
	fmt.Printf("\r%s\r", strings.Repeat(" ", clearLength)) // Clear the line

	messageColor.Print(message) // Reprint message
	boldGreen.Println(" ... Done!")    // Print " ... Done!" in bold green
}


// Helper function for the "who made you" response to keep it DRY
func printCreatorResponse() {
	fmt.Println("")
	cyan.Println("I was made for ethical hacking, cybersecurity automation, and command-line domination.")
	green.Println("I'm here to learn, adapt, and execute powerful tools like Nmap, Metasploit, Wireshark, and more — all through simple, natural language.")
	fmt.Print("If hacking is an art, ")
	magenta.Add(color.Bold).Print("Rajat")
	fmt.Println(" is the artist.")
	yellow.Add(color.Underline).Println("And I'm his masterpiece.")
	fmt.Println("")
}

func main() {
	var userInput = ""
	var lastResponse = ""
	var executablePath = ""
	var provider *string
	var apiModel *string
	var apiKey *string
	var temperature *string
	var top_p *string
	var max_length *string
	var preprompt *string
	var url *string
	var logFile *string
	var shouldExecuteCommand *bool
	var out *string
	var height *int
	var width *int
	var imgNegative *string
	var imgCount *string
	var imgRatio *string

	execPath, err := os.Executable()
	if err == nil {
		executablePath = execPath
	}
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-terminate
		boldYellow.Println("\nExiting gracefully...")
		os.Exit(0)
	}()

	args := os.Args

	apiModel = flag.String("model", "", "Choose which model to use")
	provider = flag.String("provider", os.Getenv("AI_PROVIDER"), "Choose which provider to use")
	apiKey = flag.String("key", os.Getenv("AI_API_KEY"), "Use personal API Key")
	temperature = flag.String("temperature", "", "Set temperature")
	top_p = flag.String("top_p", "", "Set top_p")
	max_length = flag.String("max_length", "", "Set max length of response")
	preprompt = flag.String("preprompt", "", "Set preprompt")

	out = flag.String("out", "", "Output file path")
	width = flag.Int("width", 1024, "Output image width")
	height = flag.Int("height", 1024, "Output image height")

	imgNegative = flag.String("img_negative", "", "Negative prompt. Avoid generating specific elements or characteristics")
	imgCount = flag.String("img_count", "1", "Number of images you want to generate")
	imgRatio = flag.String("img_ratio", "1:1", "Image Aspect Ratio")

	defaultUrl := ""

	if *provider == "openai" {
		defaultUrl = "https://api.openai.com/v1/chat/completions"
	}

	url = flag.String("url", defaultUrl, "url for openai providers")

	logFile = flag.String("log", "", "Filepath to log conversation to.")
	shouldExecuteCommand = flag.Bool(("y"), false, "Instantly execute the shell command")

	isQuiet := flag.Bool("q", false, "Gives response back without loading animation")
	flag.BoolVar(isQuiet, "quiet", false, "Gives response back without loading animation")

	isWhole := flag.Bool("w", false, "Gives response back as a whole text")
	flag.BoolVar(isWhole, "whole", false, "Gives response back as a whole text")

	isCode := flag.Bool("c", false, "Generate Code. (Experimental)")
	flag.BoolVar(isCode, "code", false, "Generate Code. (Experimental)")

	isShell := flag.Bool("s", false, "Generate and Execute shell commands.")
	flag.BoolVar(isShell, "shell", false, "Generate and Execute shell commands.")

	isImage := flag.Bool("img", false, "Generate images from text")
	flag.BoolVar(isImage, "image", false, "Generate images from text")

	isInteractive := flag.Bool("i", false, "Start normal interactive mode")
	flag.BoolVar(isInteractive, "interactive", false, "Start normal interactive mode")

	isMultiline := flag.Bool("m", false, "Start multi-line interactive mode")
	flag.BoolVar(isMultiline, "multiline", false, "Start multi-line interactive mode")

	isInteractiveShell := flag.Bool("is", false, "Start shell interactive mode")
	flag.BoolVar(isInteractiveShell, "interactive-shell", false, "Start shell interactive mode")

	isVersion := flag.Bool("v", false, "Gives response back as a whole text")
	flag.BoolVar(isVersion, "version", false, "Gives response back as a whole text")

	isHelp := flag.Bool("h", false, "Gives response back as a whole text")
	flag.BoolVar(isHelp, "help", false, "Gives response back as a whole text")

	isUpdate := flag.Bool("u", false, "Update program")
	flag.BoolVar(isUpdate, "update", false, "Update program")

	isChangelog := flag.Bool("cl", false, "See changelog of versions")
	flag.BoolVar(isChangelog, "changelog", false, "See changelog of versions")

	flag.Parse()

	// Add the "Made By Rajat" watermark and a simple logo with more colors
	magenta.Println("----------------------------------------")
	fmt.Print("|")
	cyan.Print("                                        ")
	magenta.Println("|")
	fmt.Print("|")
	yellow.Print("   _    _                                 ")
	magenta.Println("|")
	fmt.Print("|")
	yellow.Print("  | |  | |                                ")
	magenta.Println("|")
	fmt.Print("|")
	yellow.Print("  | |__| | ")
	white.Print("Made By ")
	green.Add(color.Bold).Print("Rajat")
	yellow.Print("                 ")
	magenta.Println("|")
	fmt.Print("|")
	yellow.Print("  |  __  |                                ")
	magenta.Println("|")
	fmt.Print("|")
	yellow.Print("  | |  | |                                ")
	magenta.Println("|")
	fmt.Print("|")
	yellow.Print("  |_|  |_|                                ")
	magenta.Println("|")
	fmt.Print("|")
	cyan.Print("                                        ")
	magenta.Println("|")
	magenta.Println("----------------------------------------")
	fmt.Println("") // Add a newline for better spacing

	// New "AI Power" Logo
	blue.Println("========================================")
	fmt.Print("||  "); yellow.Print("    A"); white.Print("I "); green.Print("P"); white.Print("o"); magenta.Print("w"); white.Print("e"); blue.Print("r"); white.Print("  ENABLED"); blue.Println("   ||")
	fmt.Print("||  "); magenta.Print(" \\"); white.Print(" ("); yellow.Print("_"); white.Print(") /   "); cyan.Print(" Interactive CLI "); blue.Println(" ||")
	fmt.Print("||  "); green.Print("(_("); white.Print("(_)"); blue.Print("_)"); white.Print(")    "); cyan.Print("    Assistant    "); blue.Println(" ||")
	blue.Println("========================================")
	fmt.Println("")


	main_params := structs.Params{
		ApiKey:       *apiKey,
		ApiModel:     *apiModel,
		Provider:     *provider,
		Temperature:  *temperature,
		Top_p:        *top_p,
		Max_length:   *max_length,
		Preprompt:    *preprompt,
		ThreadID:     "",
		Url:          *url,
		PrevMessages: "",
	}

	image_params := structs.ImageParams{
		ImgRatio:          *imgRatio,
		ImgNegativePrompt: *imgNegative,
		ImgCount:          *imgCount,
		Width:             *width,
		Height:            *height,
		Out:               *out,
		Params:            main_params,
	}

	prompt := flag.Arg(0)

	pipedInput := ""
	cleanPipedInput := ""
	contextText := ""

	stat, err := os.Stdin.Stat()

	if err != nil {
		utils.PrintError(fmt.Sprintf("Error accessing standard input: %v", err))
		return
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			pipedInput += scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			utils.PrintError(fmt.Sprintf("Error reading standard input: %v", err))
			return
		}
	}
	contextTextByte, _ := json.Marshal("\n\nHere is text for the context:\n")

	if len(pipedInput) > 0 {
		cleanPipedInputByte, err := json.Marshal(pipedInput)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error marshaling piped input to JSON: %v", err))
			return
		}
		cleanPipedInput = string(cleanPipedInputByte)
		cleanPipedInput = cleanPipedInput[1 : len(cleanPipedInput)-1]

		safePipedBytes, err := json.Marshal(pipedInput + "\n")
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error marshaling piped input to JSON: %v", err))
			return
		}
		pipedInput = string(safePipedBytes)
		pipedInput = pipedInput[1 : len(pipedInput)-1]
		contextText = string(contextTextByte)
	}

	if len(*preprompt) > 0 {
		*preprompt += "\n"
	}

	if len(args) > 1 {
		switch {

		case *isVersion:
			green.Print("tgpt version: ")
			yellow.Println(localVersion)
		case *isChangelog:
			helper.GetVersionHistory()
		case *isImage:
			if !*isQuiet { showLoading("Generating image") }
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -img "cat"`)
					return
				}
				imagegen.GenerateImg(trimmedPrompt, image_params, *isQuiet)
			} else {
				formattedInput := bubbletea.GetFormattedInputStdin()
				if !*isQuiet { fmt.Println() }
				imagegen.GenerateImg(formattedInput, image_params, *isQuiet)
			}
		case *isWhole:
			if !*isQuiet { showLoading("Getting whole response") }
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -w "What is encryption?"`)
					return
				}
				helper.GetWholeText(
					*preprompt+trimmedPrompt+contextText+pipedInput,
					structs.ExtraOptions{IsGetWhole: *isWhole},
					main_params,
				)
			} else {
				formattedInput := bubbletea.GetFormattedInputStdin()
				helper.GetWholeText(
					*preprompt+formattedInput+cleanPipedInput,
					structs.ExtraOptions{IsGetWhole: *isWhole},
					main_params,
				)
			}
		case *isQuiet:
			// No loading message for quiet mode by design. If one is desired, add it here.
			// showLoading("Processing request")
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -q "What is encryption?"`)
					return
				}
				helper.MakeRequestAndGetData(*preprompt+trimmedPrompt+contextText+pipedInput, main_params, structs.ExtraOptions{IsGetSilent: true})
			} else {
				formattedInput := bubbletea.GetFormattedInputStdin()
				// fmt.Println() // No extra newline if quiet
				helper.MakeRequestAndGetData(*preprompt+formattedInput+cleanPipedInput, main_params, structs.ExtraOptions{IsGetSilent: true})
			}
		case *isShell:
			if !*isQuiet { showLoading("Generating shell command") }
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -s "How to update system"`)
					return
				}
				helper.ShellCommand(
					*preprompt+trimmedPrompt+contextText+pipedInput,
					main_params,
					structs.ExtraOptions{
						IsGetCommand: true,
						AutoExec:     *shouldExecuteCommand,
					},
				)
			} else {
				utils.PrintError("You need to provide some text")
				utils.PrintError(`Example: tgpt -s "How to update system"`)
				return
			}

		case *isCode:
			if !*isQuiet { showLoading("Generating code") }
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -c "Hello world in Python"`)
					os.Exit(1)
				}
				helper.CodeGenerate(
					*preprompt+trimmedPrompt+contextText+pipedInput,
					main_params,
				)
			} else {
				utils.PrintError("You need to provide some text")
				utils.PrintError(`Example: tgpt -c "Hello world in Python"`)
				return
			}
		case *isUpdate:
			helper.Update(localVersion, executablePath)
		case *isInteractive:
			boldGreen.Print("Interactive mode started. Press Ctrl + C or type ")
			boldYellow.Print("exit")
			boldGreen.Println(" to quit.\n")

			previousMessages := ""
			threadID := utils.RandomString(36)
			history := []string{}

			getAndPrintResponse := func(input string) {
				input = strings.TrimSpace(input)
				if len(input) <= 1 {
					return
				}
				if input == "exit" {
					boldYellow.Println("Exiting interactive mode...")
					if runtime.GOOS != "windows" {
						rawModeOff := exec.Command("stty", "-raw", "echo")
						rawModeOff.Stdin = os.Stdin
						_ = rawModeOff.Run()
						rawModeOff.Wait()
					}
					os.Exit(0)
				}

				if strings.ToLower(input) == "who made you" || strings.ToLower(input) == "who created you" {
					printCreatorResponse()
					return
				}

				if len(*logFile) > 0 {
					utils.LogToFile(input, "USER_QUERY", *logFile)
				}
				if previousMessages == "" {
					input = *preprompt + input
				}

				main_params.PrevMessages = previousMessages
				main_params.ThreadID = threadID

				if !*isQuiet { showLoading("Thinking") }
				responseJson, responseTxt := helper.GetData(input, main_params, structs.ExtraOptions{IsInteractive: true, IsNormal: true})
				if len(*logFile) > 0 {
					utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
				}
				previousMessages += responseJson
				history = append(history, input)
				lastResponse = responseTxt
			}

			input := strings.TrimSpace(prompt)
			if len(input) > 1 {
				cyan.Print("╭─ ")
				boldWhite.Println("You")
				magenta.Print("╰─> ")
				fmt.Println(input) // Display the initial prompt text
				getAndPrintResponse(input)
			}

			for {
				cyan.Print("╭─ ")
				boldWhite.Println("You")
				input := Prompt.Input("╰─> ", bubbletea.HistoryCompleter,
					Prompt.OptionHistory(history),
					Prompt.OptionPrefixTextColor(Prompt.Blue), // Color for "╰─> "
					Prompt.OptionInputTextColor(Prompt.White),     // Color for user typed text
					Prompt.OptionAddKeyBind(Prompt.KeyBind{
						Key: Prompt.ControlC,
						Fn:  exit,
					}),
				)
				getAndPrintResponse(input)
			}

		case *isMultiline:
			boldGreen.Print("\nMultiline Interactive mode. Ctrl+D to submit, Ctrl+C to exit.")
			fmt.Print(" Esc to unfocus, i to focus. Unfocused: p=paste, c=copy response, b=copy last code block.\n")

			previousMessages := ""
			threadID := utils.RandomString(36)

			for programLoop {
				fmt.Print("\n")
				p := tea.NewProgram(bubbletea.InitialModel(preprompt, &programLoop, &lastResponse, &userInput))
				_, err := p.Run()

				if err != nil {
					utils.PrintError(err.Error())
					os.Exit(1)
				}
				if len(userInput) > 0 {
					if strings.ToLower(userInput) == "who made you" || strings.ToLower(userInput) == "who created you" {
						printCreatorResponse()
						userInput = "" // Clear userInput to prevent processing by AI
						continue
					}

					if len(*logFile) > 0 {
						utils.LogToFile(userInput, "USER_QUERY", *logFile)
					}

					main_params.PrevMessages = previousMessages
					main_params.ThreadID = threadID

					if !*isQuiet { showLoading("Thinking") }
					responseJson, responseTxt := helper.GetData(userInput, main_params, structs.ExtraOptions{IsInteractive: true, IsNormal: true})
					previousMessages += responseJson
					lastResponse = responseTxt

					if len(*logFile) > 0 {
						utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
					}
					userInput = "" // Clear user input after processing
				}
			}

		case *isInteractiveShell:
			boldGreen.Print("Interactive Shell mode. Ctrl+C or type ")
			boldYellow.Print("exit")
			boldGreen.Println(" to quit.\n")

			helper.SetShellAndOSVars()
			promptIs := fmt.Sprintf("You are a powerful terminal assistant. Answer the needs of the user."+
				"You can execute command in command line if need. Always wrap the command with the xml tag `<cmd>`."+
				"Only output command when you think user wants to execute a command. Execute only one command in one response."+
				"The shell environment you are is %s. The operate system you are is %s."+
				"Examples:"+
				"User: list the files in my home dir."+
				"Assistant: Sure. I will list the files under your home dir. <cmd>ls ~</cmd>",
				helper.ShellName, helper.OperatingSystem,
			)
			previousMessages := ""
			threadID := utils.RandomString(36)
			history := []string{}

			getAndPrintResponse := func(input string) string {
				input = strings.TrimSpace(input)
				if len(input) <= 1 {
					return ""
				}
				if input == "exit" {
					boldYellow.Println("Exiting interactive shell...")
					if runtime.GOOS != "windows" {
						rawModeOff := exec.Command("stty", "-raw", "echo")
						rawModeOff.Stdin = os.Stdin
						_ = rawModeOff.Run()
						rawModeOff.Wait()
					}
					os.Exit(0)
				}

				if strings.ToLower(input) == "who made you" || strings.ToLower(input) == "who created you" {
					printCreatorResponse()
					return ""
				}

				if len(*logFile) > 0 {
					utils.LogToFile(input, "USER_QUERY", *logFile)
				}
				if previousMessages == "" {
					input = *preprompt + input
				}

				main_params.PrevMessages = previousMessages
				main_params.ThreadID = threadID
				main_params.SystemPrompt = promptIs

				if !*isQuiet { showLoading("Processing command") }
				responseJson, responseTxt := helper.GetData(input, main_params, structs.ExtraOptions{IsInteractiveShell: true, IsNormal: true})
				commandRegex := regexp.MustCompile(`<cmd>(.*?)</cmd>`)
				matches := commandRegex.FindStringSubmatch(responseTxt)
				if len(matches) > 1 {
					command := strings.TrimSpace(matches[1])
					return command
				}
				if len(*logFile) > 0 {
					utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
				}
				previousMessages += responseJson
				history = append(history, input)
				lastResponse = responseTxt
				return ""
			}

			execCmd := func(cmd string) {
				if cmd != "" {
					if *shouldExecuteCommand {
						fmt.Println()
						helper.ExecuteCommand(helper.ShellName, helper.ShellOptions, cmd)
					} else {
						fmt.Println() // Newline before the confirmation
						boldYellow.Printf("Execute shell command: ")
						white.Printf("`%s`", cmd)
						boldYellow.Print(" ? [")
						green.Print("y")
						boldYellow.Print("/")
						red.Print("n")
						boldYellow.Print("]: ")

						userInput := Prompt.Input("", bubbletea.HistoryCompleter, // No prefix text needed here
							Prompt.OptionPrefixTextColor(Prompt.Blue), // Not visible due to empty prefix
							Prompt.OptionInputTextColor(Prompt.White),
							Prompt.OptionAddKeyBind(Prompt.KeyBind{
								Key: Prompt.ControlC,
								Fn:  exit,
							}),
						)
						userInput = strings.TrimSpace(strings.ToLower(userInput))

						if userInput == "y" || userInput == "" {
							green.Println("Executing...")
							helper.ExecuteCommand(helper.ShellName, helper.ShellOptions, cmd)
						} else {
							red.Println("Execution cancelled.")
						}
					}
				}
			}

			input := strings.TrimSpace(prompt)
			if len(input) > 1 {
				cyan.Print("╭─ ")
				boldWhite.Println("You")
				magenta.Print("╰─> ")
				fmt.Println(input) // Display the initial prompt text
				cmd := getAndPrintResponse(input)
				execCmd(cmd)
			}

			for {
				cyan.Print("╭─ ")
				boldWhite.Println("You")
				input := Prompt.Input("╰─> ", bubbletea.HistoryCompleter,
					Prompt.OptionHistory(history),
					Prompt.OptionPrefixTextColor(Prompt.Blue), // <<<< MODIFIED THIS LINE
					Prompt.OptionInputTextColor(Prompt.White),
					Prompt.OptionAddKeyBind(Prompt.KeyBind{
						Key: Prompt.ControlC,
						Fn:  exit,
					}),
				)
				cmd := getAndPrintResponse(input)
				execCmd(cmd)
			}

		case *isHelp:
			helper.ShowHelpMessage()
		default:
			formattedInput := strings.TrimSpace(prompt)

			if len(formattedInput) <= 1 {
				utils.PrintError("You need to write something")
				return
			}
			if strings.ToLower(formattedInput) == "who made you" || strings.ToLower(formattedInput) == "who created you" {
				printCreatorResponse()
				return
			}

			if !*isQuiet { showLoading("Processing request") }
			helper.GetData(
				*preprompt+formattedInput+contextText+pipedInput,
				main_params,
				structs.ExtraOptions{
					IsNormal: true, IsInteractive: false,
				})
		}

	} else { // Handle case with no flags, just direct input or piped input
		var allInput string
		if len(pipedInput) > 0 { // Prefer piped input if available
			allInput = pipedInput
		} else {
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				allInput = scanner.Text()
			}
			if err := scanner.Err(); err != nil {
				utils.PrintError(fmt.Sprintf("Error reading standard input: %v", err))
				return
			}
		}

		formattedInput := strings.TrimSpace(allInput)
		if len(formattedInput) == 0 && len(cleanPipedInput) == 0 { // Check if any input was actually provided
			utils.PrintError("No input provided. Pipe text or type your query.")
			helper.ShowHelpMessage() // Show help if no input at all
			return
		}


		if strings.ToLower(formattedInput) == "who made you" || strings.ToLower(formattedInput) == "who created you" {
			printCreatorResponse()
			return
		}

		finalQuery := ""
		if len(cleanPipedInput) > 0 { // If there was piped input, use it (cleanPipedInput has original formatting)
			finalQuery = *preprompt + formattedInput + contextText + cleanPipedInput
		} else {
			finalQuery = *preprompt + formattedInput
		}


		if !*isQuiet { showLoading("Processing request") }
		helper.GetData(finalQuery, main_params, structs.ExtraOptions{IsInteractive: false})
	}
}

func exit(_ *Prompt.Buffer) {
	boldYellow.Println("\nExiting...") // Use a colored exit message

	if runtime.GOOS != "windows" {
		rawModeOff := exec.Command("stty", "-raw", "echo")
		rawModeOff.Stdin = os.Stdin
		_ = rawModeOff.Run()
		rawModeOff.Wait()
	}
	os.Exit(0)
}