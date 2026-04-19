package main

import (
	"fmt"
	"io"
)

// cmdCompletions prints a static completion script for the requested
// shell. Static scripts don't need to invoke the binary at tab-time,
// which keeps things fast and predictable.
func cmdCompletions(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "error: specify a shell — bash, zsh, or fish")
		return 1
	}
	switch args[0] {
	case "bash":
		fmt.Fprint(stdout, bashCompletion)
	case "zsh":
		fmt.Fprint(stdout, zshCompletion)
	case "fish":
		fmt.Fprint(stdout, fishCompletion)
	default:
		fmt.Fprintf(stderr, "error: unknown shell %q (expected: bash, zsh, fish)\n", args[0])
		return 1
	}
	return 0
}

const bashCompletion = `# sendy bash completion — source or copy into /etc/bash_completion.d/
_sendy() {
    local cur prev subcmd
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    subcmd="${COMP_WORDS[1]}"

    if [ "$COMP_CWORD" -eq 1 ]; then
        COMPREPLY=( $(compgen -W "create list ls view cat raw login logout whoami claim completions help version" -- "$cur") )
        return
    fi

    case "$subcmd" in
        create)
            case "$prev" in
                --password|--user-key) return ;;
            esac
            if [[ "$cur" == -* ]]; then
                COMPREPLY=( $(compgen -W "--password --user-key" -- "$cur") )
            else
                COMPREPLY=( $(compgen -f -- "$cur") )
            fi
            ;;
        list|ls)
            COMPREPLY=( $(compgen -W "--limit --offset --user-key --search" -- "$cur") )
            ;;
        view|cat)
            case "$prev" in
                --password) return ;;
            esac
            [[ "$cur" == -* ]] && COMPREPLY=( $(compgen -W "--password" -- "$cur") )
            ;;
        claim)
            COMPREPLY=( $(compgen -W "--user-key" -- "$cur") )
            ;;
        completions)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- "$cur") )
            ;;
    esac
}
complete -F _sendy sendy
`

const zshCompletion = `#compdef sendy
# sendy zsh completion — drop into a dir on $fpath (e.g. ~/.zsh/completions)

_sendy() {
    local -a commands
    commands=(
        'create:Create a paste from file or stdin'
        'list:List your pastes'
        'ls:Alias for list'
        'view:Print a paste to stdout'
        'cat:Alias for view'
        'raw:Print raw paste text'
        'login:Open browser to sign in'
        'logout:Remove stored token'
        'whoami:Show current identity'
        'claim:Claim anonymous pastes under signed-in account'
        'completions:Generate shell completion script'
        'help:Show help'
        'version:Print version'
    )

    if (( CURRENT == 2 )); then
        _describe 'sendy command' commands
        return
    fi

    case "$words[2]" in
        create)
            _arguments \
                '--password[Password-protect the paste]:password:' \
                '--user-key[Override user_key]:key:' \
                '*:file:_files'
            ;;
        list|ls)
            _arguments \
                '--limit[Max pastes to return]:n:' \
                '--offset[Pagination offset]:n:' \
                '--user-key[Override SENDY_USER_KEY]:key:' \
                '--search[Filter by substring]:query:'
            ;;
        view|cat)
            _arguments \
                '--password[Unlock password-protected paste]:password:' \
                '*:slug:'
            ;;
        claim)
            _arguments '--user-key[user_key to claim]:key:'
            ;;
        completions)
            _values 'shell' bash zsh fish
            ;;
    esac
}

_sendy "$@"
`

const fishCompletion = `# sendy fish completion — save as ~/.config/fish/completions/sendy.fish

complete -c sendy -f

complete -c sendy -n '__fish_use_subcommand' -a create -d 'Create a paste from file or stdin'
complete -c sendy -n '__fish_use_subcommand' -a list -d 'List your pastes'
complete -c sendy -n '__fish_use_subcommand' -a ls -d 'Alias for list'
complete -c sendy -n '__fish_use_subcommand' -a view -d 'Print a paste to stdout'
complete -c sendy -n '__fish_use_subcommand' -a cat -d 'Alias for view'
complete -c sendy -n '__fish_use_subcommand' -a raw -d 'Print raw paste text'
complete -c sendy -n '__fish_use_subcommand' -a login -d 'Open browser to sign in'
complete -c sendy -n '__fish_use_subcommand' -a logout -d 'Remove stored token'
complete -c sendy -n '__fish_use_subcommand' -a whoami -d 'Show current identity'
complete -c sendy -n '__fish_use_subcommand' -a claim -d 'Claim anonymous pastes'
complete -c sendy -n '__fish_use_subcommand' -a completions -d 'Generate shell completion script'
complete -c sendy -n '__fish_use_subcommand' -a help -d 'Show help'
complete -c sendy -n '__fish_use_subcommand' -a version -d 'Print version'

complete -c sendy -n '__fish_seen_subcommand_from create' -l password -d 'Password-protect the paste' -r
complete -c sendy -n '__fish_seen_subcommand_from create' -l user-key -d 'Override user_key' -r
complete -c sendy -n '__fish_seen_subcommand_from create' -F

complete -c sendy -n '__fish_seen_subcommand_from list ls' -l limit -d 'Max pastes to return' -r
complete -c sendy -n '__fish_seen_subcommand_from list ls' -l offset -d 'Pagination offset' -r
complete -c sendy -n '__fish_seen_subcommand_from list ls' -l user-key -d 'Override SENDY_USER_KEY' -r
complete -c sendy -n '__fish_seen_subcommand_from list ls' -l search -d 'Filter by substring' -r

complete -c sendy -n '__fish_seen_subcommand_from view cat' -l password -d 'Unlock password-protected paste' -r

complete -c sendy -n '__fish_seen_subcommand_from claim' -l user-key -d 'user_key to claim' -r

complete -c sendy -n '__fish_seen_subcommand_from completions' -a 'bash zsh fish'
`
