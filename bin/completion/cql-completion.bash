#!bash
#
# bash completion for cql (http://covenantsql.io)
#

_cql_comp()
{
    COMPREPLY=( $(compgen -W "$1" -- ${word}) )
}

_cql_help_generate()
{
    opts=$(cql generate -help 2>&1 | grep '  -' | sed 's/  //' | cut -d ' ' -f 1)
    _cql_comp "config public ${opts//\\n/ }"
}

_cql_help_console()
{
    opts=$(cql console -help 2>&1 | grep '  -' | sed 's/  //' | cut -d ' ' -f 1)
    _cql_comp "${opts//\\n/ }"
}

_cql_help_create()
{
    opts=$(cql create -help 2>&1 | grep '  -' | sed 's/  //' | cut -d ' ' -f 1)
    _cql_comp "${opts//\\n/ }"
}

_cql_help_drop()
{
    opts=$(cql drop -help 2>&1 | grep '  -' | sed 's/  //' | cut -d ' ' -f 1)
    _cql_comp "covenantsql:// ${opts//\\n/ }"
}

_cql_help_wallet()
{
    opts=$(cql wallet -help 2>&1 | grep '  -' | sed 's/  //' | cut -d ' ' -f 1)
    _cql_comp "${opts//\\n/ }"
}

_cql_help_transfer()
{
    opts=$(cql transfer -help 2>&1 | grep '  -' | sed 's/  //' | cut -d ' ' -f 1)
    _cql_comp "${opts//\\n/ }"
}

_cql_help_grant()
{
    opts=$(cql grant -help 2>&1 | grep '  -' | sed 's/  //' | cut -d ' ' -f 1)
    _cql_comp "${opts//\\n/ }"
}

_cql_help_mirror()
{
    opts=$(cql mirror -help 2>&1 | grep '  -' | sed 's/  //' | cut -d ' ' -f 1)
    _cql_comp "${opts//\\n/ }"
}

_cql_help_explorer()
{
    opts=$(cql explorer -help 2>&1 | grep '  -' | sed 's/  //' | cut -d ' ' -f 1)
    _cql_comp "${opts//\\n/ }"
}

_cql_help_adapter()
{
    opts=$(cql adapter -help 2>&1 | grep '  -' | sed 's/  //' | cut -d ' ' -f 1)
    _cql_comp "${opts//\\n/ }"
}

_cql_help_idminer()
{
    opts=$(cql idminer -help 2>&1 | grep '  -' | sed 's/  //' | cut -d ' ' -f 1)
    _cql_comp "${opts//\\n/ }"
}

_cql_help_rpc()
{
    opts=$(cql rpc -help 2>&1 | grep '  -' | sed 's/  //' | cut -d ' ' -f 1)
    _cql_comp "${opts//\\n/ }"
}

_cql_help_version()
{
    return
}

_cql()
{
    COMPREPLY=()

    local word="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"

    case "${COMP_CWORD}" in
        1)
            local opts="generate console create drop wallet transfer \
              grant mirror explorer adapter idminer rpc version"
            COMPREPLY=( $(compgen -W "${opts}" -- ${word}) );;
        *)
            local command="${COMP_WORDS[1]}"
            eval "_cql_help_$command" 2> /dev/null ;;
#        *)
#            local command="${COMP_WORDS[1]}"
#            local subcommand="${COMP_WORDS[2]}"
#            eval "_cql_${command}_${subcommand}" 2> /dev/null && return
#            eval "_cql_$command" 2> /dev/null ;;
    esac
}

complete -F _cql cql