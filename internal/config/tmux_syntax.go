package config

import (
	"fmt"
	"strings"
)

type sourceDirective struct {
	Paths         []string
	Optional      bool
	ExpandFormats bool
	ParseOnly     bool
	Line          int
	Text          string
}

type locatedSourceDirective struct {
	sourceDirective
	start int
	end   int
}

// ConfigParseError identifies malformed syntax in a tmux configuration.
type ConfigParseError struct {
	Path string
	Line int
	Msg  string
}

func (e *ConfigParseError) Error() string {
	return fmt.Sprintf("%s:%d: %s", e.Path, e.Line, e.Msg)
}

type tmuxToken struct {
	value string
	start int
	end   int
	line  int
	plain bool
}

type tmuxCommand struct {
	tokens []tmuxToken
}

type tmuxScanOptions struct {
	commandSemicolons bool
	commandNewlines   bool
	commentAnywhere   bool
	lineContinuations bool
	rejectBraces      bool
}

const (
	unmatchedSingleQuote = "unmatched single quote"
	unmatchedDoubleQuote = "unmatched double quote"
	conditionalIf        = "%if"
	conditionalElif      = "%elif"
	conditionalElse      = "%else"
	conditionalEndif     = "%endif"
	hiddenDirective      = "%hidden"
	ifShellCommand       = "if-shell"
)

func splitTmuxCommandLine(line string) ([]string, bool) {
	commands, err := scanTmuxSyntax("", line, tmuxScanOptions{})
	if err != nil || len(commands) == 0 {
		return nil, err == nil
	}
	args := make([]string, 0, len(commands[0].tokens))
	for _, token := range commands[0].tokens {
		args = append(args, token.value)
	}
	return args, true
}

func scanTmuxCommands(path, content string, rejectBraces bool) ([]tmuxCommand, error) {
	return scanTmuxSyntax(path, content, tmuxScanOptions{
		commandSemicolons: true,
		commandNewlines:   true,
		commentAnywhere:   true,
		lineContinuations: true,
		rejectBraces:      rejectBraces,
	})
}

func scanTmuxSyntax(path, content string, options tmuxScanOptions) ([]tmuxCommand, error) {
	scanner := tmuxScanner{
		path:    path,
		content: content,
		options: options,
		line:    1,
	}
	return scanner.scan()
}

type tmuxScanner struct {
	path    string
	content string
	options tmuxScanOptions

	commands   []tmuxCommand
	tokens     []tmuxToken
	word       strings.Builder
	line       int
	quote      byte
	quoteLine  int
	inWord     bool
	tokenStart int
	tokenEnd   int
	tokenLine  int
	tokenPlain bool
	inComment  bool
}

func (s *tmuxScanner) scan() ([]tmuxCommand, error) {
	for at := 0; at < len(s.content); {
		next, err := s.consume(at)
		if err != nil {
			return nil, err
		}
		at = next
	}
	if s.quote != 0 {
		return nil, configSyntaxError(s.path, s.quoteLine, unmatchedQuoteMessage(s.quote))
	}
	s.flushCommand()
	return s.commands, nil
}

func (s *tmuxScanner) consume(at int) (int, error) {
	if next, handled, err := s.consumeLineContinuation(at); handled || err != nil {
		return next, err
	}
	if s.inComment {
		return s.consumeComment(at), nil
	}
	if s.quote != 0 {
		return s.consumeQuoted(at)
	}
	if s.startsConditionalFormat(at) {
		return s.consumeConditionalFormat(at)
	}
	if s.options.rejectBraces && !s.inWord && (s.content[at] == '{' || s.content[at] == '}') {
		return at, configSyntaxError(s.path, s.line, "unquoted braced command lists are unsupported")
	}
	return s.consumeUnquoted(at), nil
}

func (s *tmuxScanner) startsConditionalFormat(at int) bool {
	return at+1 < len(s.content) && s.content[at] == '#' && s.content[at+1] == '{' &&
		s.expectsConditionalFormat()
}

func (s *tmuxScanner) consumeConditionalFormat(at int) (int, error) {
	startLine := s.line
	s.startWord(at, true)
	depth := 0
	for cursor := at; cursor < len(s.content); {
		if s.options.lineContinuations && s.content[cursor] == '\\' &&
			cursor+1 < len(s.content) && s.content[cursor+1] == '\n' {
			s.tokenEnd = cursor + 2
			s.line++
			cursor += 2
			continue
		}
		if cursor+1 < len(s.content) && s.content[cursor] == '#' && s.content[cursor+1] == '{' {
			s.word.WriteString("#{")
			depth++
			cursor += 2
			s.tokenEnd = cursor
			continue
		}

		char := s.content[cursor]
		s.word.WriteByte(char)
		s.tokenEnd = cursor + 1
		cursor++
		if char == '\n' {
			s.line++
		}
		if char == '}' {
			depth--
			if depth == 0 {
				return cursor, nil
			}
		}
	}
	return at, configSyntaxError(s.path, startLine, "unmatched tmux format expression")
}

func (s *tmuxScanner) consumeLineContinuation(at int) (next int, handled bool, err error) {
	if !s.options.lineContinuations || s.content[at] != '\\' {
		return at, false, nil
	}
	if at+1 >= len(s.content) {
		return at, true, configSyntaxError(s.path, s.line, "dangling line continuation")
	}
	if s.content[at+1] != '\n' {
		return at, false, nil
	}
	if s.inWord {
		s.tokenEnd = at + 2
	}
	s.line++
	return at + 2, true, nil
}

func (s *tmuxScanner) consumeComment(at int) int {
	if s.content[at] == '\n' {
		s.inComment = false
		s.line++
	}
	return at + 1
}

func (s *tmuxScanner) consumeQuoted(at int) (int, error) {
	char := s.content[at]
	if char == '\n' && s.options.commandNewlines {
		return at, configSyntaxError(s.path, s.quoteLine, unmatchedQuoteMessage(s.quote))
	}
	if char == s.quote {
		s.quote = 0
		s.tokenEnd = at + 1
		return at + 1, nil
	}
	if char == '\\' && s.quote == '"' && at+1 < len(s.content) {
		return s.consumeEscaped(at), nil
	}
	s.word.WriteByte(char)
	s.tokenEnd = at + 1
	if char == '\n' {
		s.line++
	}
	return at + 1, nil
}

func (s *tmuxScanner) consumeUnquoted(at int) int {
	char := s.content[at]
	switch {
	case char == '\'' || char == '"':
		s.startWord(at, false)
		s.quote = char
		s.quoteLine = s.line
		s.tokenEnd = at + 1
		return at + 1
	case s.startsComment(at):
		s.flushCommand()
		s.inComment = true
		return at + 1
	case char == '\\' && at+1 < len(s.content):
		return s.consumeEscaped(at)
	case char == ';' && s.options.commandSemicolons:
		s.flushCommand()
		return at + 1
	case char == '\n' && s.options.commandNewlines:
		s.flushCommand()
		s.line++
		return at + 1
	case isTmuxWhitespace(char):
		s.flushWord()
		if char == '\n' {
			s.line++
		}
		return at + 1
	default:
		s.startWord(at, true)
		s.word.WriteByte(char)
		s.tokenEnd = at + 1
		return at + 1
	}
}

func (s *tmuxScanner) consumeEscaped(at int) int {
	s.startWord(at, false)
	s.word.WriteByte(s.content[at+1])
	s.tokenEnd = at + 2
	if s.content[at+1] == '\n' {
		s.line++
	}
	return at + 2
}

func (s *tmuxScanner) startsComment(at int) bool {
	if s.content[at] != '#' {
		return false
	}
	return s.options.commentAnywhere || !s.inWord
}

func (s *tmuxScanner) expectsConditionalFormat() bool {
	if !s.options.commentAnywhere || s.inWord || len(s.tokens) == 0 {
		return false
	}
	previous := s.tokens[len(s.tokens)-1]
	return previous.plain && (previous.value == conditionalIf || previous.value == conditionalElif)
}

func (s *tmuxScanner) startWord(at int, plain bool) {
	if !s.inWord {
		s.inWord = true
		s.tokenStart = at
		s.tokenLine = s.line
		s.tokenPlain = true
	}
	if !plain {
		s.tokenPlain = false
	}
}

func (s *tmuxScanner) flushWord() {
	if !s.inWord {
		return
	}
	s.tokens = append(s.tokens, tmuxToken{
		value: s.word.String(),
		start: s.tokenStart,
		end:   s.tokenEnd,
		line:  s.tokenLine,
		plain: s.tokenPlain,
	})
	s.word.Reset()
	s.inWord = false
}

func (s *tmuxScanner) flushCommand() {
	s.flushWord()
	if len(s.tokens) == 0 {
		return
	}
	s.commands = append(s.commands, tmuxCommand{tokens: s.tokens})
	s.tokens = nil
}

func unmatchedQuoteMessage(quote byte) string {
	if quote == '\'' {
		return unmatchedSingleQuote
	}
	return unmatchedDoubleQuote
}

func isTmuxWhitespace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\r' || char == '\n'
}

type conditionalFrame struct {
	parentActive bool
	branchTaken  bool
	active       bool
	seenElse     bool
	line         int
}

func parseSourceDirectives(
	file, content string,
	expandFormat func(string) (string, error),
) ([]sourceDirective, error) {
	located, err := parseLocatedSourceDirectives(file, content, expandFormat)
	if err != nil {
		return nil, err
	}
	directives := make([]sourceDirective, 0, len(located))
	for _, directive := range located {
		directives = append(directives, directive.sourceDirective)
	}
	return directives, nil
}

func parseLocatedSourceDirectives(
	file, content string,
	expandFormat func(string) (string, error),
) ([]locatedSourceDirective, error) {
	commands, err := scanTmuxCommands(file, content, true)
	if err != nil {
		return nil, err
	}

	var directives []locatedSourceDirective
	var conditionals []conditionalFrame
	for _, command := range commands {
		for at := 0; at < len(command.tokens); {
			if isConditionalToken(command.tokens[at]) {
				consumed, conditionalErr := applyConditional(
					file,
					command.tokens[at:],
					&conditionals,
					expandFormat,
				)
				if conditionalErr != nil {
					return nil, conditionalErr
				}
				at += consumed
				continue
			}

			next := at + 1
			for next < len(command.tokens) && !isConditionalToken(command.tokens[next]) {
				next++
			}
			segment := command.tokens[at:next]
			if unsupported := unsupportedTmuxDirective(segment[0]); unsupported != "" {
				return nil, configSyntaxError(file, segment[0].line, "unsupported tmux directive "+unsupported)
			}
			if !conditionalActive(conditionals) {
				at = next
				continue
			}
			if err := rejectNestedExecutableSources(file, content, segment); err != nil {
				return nil, err
			}
			if isSourceCommand(segment[0].value) {
				directive, directiveErr := parseSourceCommand(file, content, segment)
				if directiveErr != nil {
					return nil, directiveErr
				}
				directives = append(directives, locatedSourceDirective{
					sourceDirective: directive,
					start:           segment[0].start,
					end:             segment[len(segment)-1].end,
				})
			}
			at = next
		}
	}

	if len(conditionals) != 0 {
		frame := conditionals[len(conditionals)-1]
		return nil, configSyntaxError(file, frame.line, "missing %endif")
	}
	return directives, nil
}

func unsupportedTmuxDirective(token tmuxToken) string {
	if !token.plain || !strings.HasPrefix(token.value, "%") || token.value == hiddenDirective {
		return ""
	}
	return token.value
}

func isSourceCommand(command string) bool {
	return command == "source" || command == "source-file"
}

func rejectNestedExecutableSources(file, content string, tokens []tmuxToken) error {
	if len(tokens) == 0 || !tokens[0].plain || (tokens[0].value != ifShellCommand && tokens[0].value != "if") {
		return nil
	}

	at := ifShellConditionIndex(tokens)
	if at >= len(tokens) {
		return nil
	}
	for _, token := range tokens[at+1:] {
		if !isQuotedToken(content, token) || !commandListContainsSource(token.value) {
			continue
		}
		return configSyntaxError(file, tokens[0].line, "executable quoted command list may contain source/source-file")
	}
	return nil
}

func ifShellConditionIndex(tokens []tmuxToken) int {
	at := 1
	for at < len(tokens) {
		option := tokens[at].value
		if option == "--" {
			return at + 1
		}
		if len(option) < 2 || option[0] != '-' {
			return at
		}
		consumeTarget := option[len(option)-1] == 't'
		at++
		if consumeTarget {
			at++
		}
	}
	return at
}

func isQuotedToken(content string, token tmuxToken) bool {
	raw := strings.TrimSpace(content[token.start:token.end])
	if len(raw) < 2 {
		return false
	}
	return (raw[0] == '\'' && raw[len(raw)-1] == '\'') || (raw[0] == '"' && raw[len(raw)-1] == '"')
}

func commandListContainsSource(value string) bool {
	commands, err := scanTmuxCommands("", value, false)
	if err != nil {
		return containsSourceWord(value)
	}
	for _, command := range commands {
		if len(command.tokens) == 0 {
			continue
		}
		if isSourceCommand(command.tokens[0].value) {
			return true
		}
		if command.tokens[0].value != ifShellCommand && command.tokens[0].value != "if" {
			continue
		}
		conditionAt := ifShellConditionIndex(command.tokens)
		if conditionAt >= len(command.tokens) {
			continue
		}
		for _, branch := range command.tokens[conditionAt+1:] {
			if commandListContainsSource(branch.value) {
				return true
			}
		}
	}
	return false
}

func containsSourceWord(value string) bool {
	for _, field := range strings.FieldsFunc(value, func(char rune) bool {
		return char == ' ' || char == '\t' || char == '\r' || char == '\n' || char == ';'
	}) {
		if isSourceCommand(field) {
			return true
		}
	}
	return false
}

func isConditionalToken(token tmuxToken) bool {
	if !token.plain {
		return false
	}
	switch token.value {
	case conditionalIf, conditionalElif, conditionalElse, conditionalEndif:
		return true
	default:
		return false
	}
}

func conditionalActive(conditionals []conditionalFrame) bool {
	return len(conditionals) == 0 || conditionals[len(conditionals)-1].active
}

func applyConditional(
	file string,
	tokens []tmuxToken,
	conditionals *[]conditionalFrame,
	expandFormat func(string) (string, error),
) (int, error) {
	marker := tokens[0]
	if marker.value == conditionalIf {
		parentActive := conditionalActive(*conditionals)
		condition, err := conditionalValue(file, tokens, marker, expandFormat, parentActive)
		if err != nil {
			return 0, err
		}
		active := parentActive && condition
		*conditionals = append(*conditionals, conditionalFrame{
			parentActive: parentActive,
			branchTaken:  active,
			active:       active,
			line:         marker.line,
		})
		return 2, nil
	}
	if len(*conditionals) == 0 {
		return 0, configSyntaxError(file, marker.line, "unexpected "+marker.value)
	}
	frame := &(*conditionals)[len(*conditionals)-1]
	switch marker.value {
	case conditionalElif:
		if frame.seenElse {
			return 0, configSyntaxError(file, marker.line, "%elif after %else")
		}
		evaluate := frame.parentActive && !frame.branchTaken
		condition, err := conditionalValue(file, tokens, marker, expandFormat, evaluate)
		if err != nil {
			return 0, err
		}
		frame.active = evaluate && condition
		frame.branchTaken = frame.branchTaken || frame.active
		return 2, nil
	case conditionalElse:
		if frame.seenElse {
			return 0, configSyntaxError(file, marker.line, "duplicate %else")
		}
		frame.seenElse = true
		frame.active = frame.parentActive && !frame.branchTaken
		frame.branchTaken = frame.branchTaken || frame.active
		return 1, nil
	default: // The caller accepts only conditional tokens, leaving %endif.
		*conditionals = (*conditionals)[:len(*conditionals)-1]
		return 1, nil
	}
}

func conditionalValue(
	file string,
	tokens []tmuxToken,
	marker tmuxToken,
	expandFormat func(string) (string, error),
	evaluate bool,
) (bool, error) {
	if len(tokens) < 2 || isConditionalToken(tokens[1]) {
		return false, configSyntaxError(file, marker.line, marker.value+" requires a condition")
	}
	if tokens[1].plain && strings.HasPrefix(tokens[1].value, "{") {
		return false, configSyntaxError(file, marker.line, "braced conditional expressions are unsupported")
	}
	value := tokens[1].value
	hasFormats, balanced := tmuxFormatSpans(value)
	if !balanced {
		return false, configSyntaxError(file, marker.line, "unbalanced tmux format span in conditional")
	}
	if !evaluate {
		return false, nil
	}
	if hasFormats {
		expanded, err := expandFormat(value)
		if err != nil {
			return false, configSyntaxError(file, marker.line, "expand conditional format: "+err.Error())
		}
		value = expanded
	}
	return value != "" && value != "0", nil
}

func tmuxFormatSpans(value string) (hasFormats, balanced bool) {
	for at := 0; at < len(value); {
		relativeStart := strings.Index(value[at:], "#{")
		if relativeStart < 0 {
			return hasFormats, true
		}
		hasFormats = true
		cursor := at + relativeStart + 2
		depth := 1
		for cursor < len(value) && depth > 0 {
			switch {
			case cursor+1 < len(value) && value[cursor] == '#' && value[cursor+1] == '{':
				depth++
				cursor += 2
			case value[cursor] == '}':
				depth--
				cursor++
			default:
				cursor++
			}
		}
		if depth != 0 {
			return true, false
		}
		at = cursor
	}
	return hasFormats, true
}

func parseSourceCommand(file, content string, tokens []tmuxToken) (sourceDirective, error) {
	line := tokens[0].line
	for _, token := range tokens {
		raw := content[token.start:token.end]
		if token.plain && strings.HasPrefix(raw, "{") {
			return sourceDirective{}, configSyntaxError(file, line, "braced arguments are unsupported in source directives")
		}
	}

	directive := sourceDirective{Line: line}
	at := 1
	for at < len(tokens) {
		option := tokens[at].value
		if len(option) < 2 || option[0] != '-' {
			break
		}
		consumesTarget, err := applySourceOption(file, line, option, &directive)
		if err != nil {
			return sourceDirective{}, err
		}
		if consumesTarget {
			at++
			if at >= len(tokens) {
				return sourceDirective{}, configSyntaxError(file, line, "source option -t requires a value")
			}
		}
		at++
	}

	if at >= len(tokens) {
		return sourceDirective{}, configSyntaxError(file, line, "source directive requires at least one path")
	}
	directive.Paths = make([]string, 0, len(tokens)-at)
	for _, token := range tokens[at:] {
		directive.Paths = append(directive.Paths, token.value)
	}
	directive.Text = strings.TrimSpace(content[tokens[0].start:tokens[len(tokens)-1].end])
	return directive, nil
}

func applySourceOption(
	file string,
	line int,
	option string,
	directive *sourceDirective,
) (consumesTarget bool, err error) {
	for at := 1; at < len(option); at++ {
		switch option[at] {
		case 'F':
			directive.ExpandFormats = true
		case 'n':
			directive.ParseOnly = true
		case 'v':
		case 'q':
			directive.Optional = true
		case 't':
			return at+1 == len(option), nil
		default:
			return false, configSyntaxError(file, line, fmt.Sprintf("unknown source option -%c", option[at]))
		}
	}
	return false, nil
}

func configSyntaxError(path string, line int, message string) error {
	return &ConfigParseError{Path: path, Line: line, Msg: message}
}
