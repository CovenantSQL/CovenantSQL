#compdef cql
# ------------------------------------------------------------------------------
# Description
# -----------
#
#  zsh completion for cql (http://covenantsql.io)
#
# ------------------------------------------------------------------------------
# Authors
# -------
#
#  * Robinhuett <https://github.com/Robinhuett>
#  * Auxten <auxtenwpc#gmail.com>
#
# ------------------------------------------------------------------------------

_cql_dsn() {
    compadd -S '' "$@" - covenantsql://
}

_cql_args() {
  case $words[1] in
    (help)
      _arguments '*:help:(generate console create drop wallet transfer grant mirror explorer adapter idminer rpc
)'
      ;;
    (generate)
      _arguments '-config=-[Config file for CovenantSQL, default "~/.cql/config.yaml]:CONF file:_files' \
        '-bypass-signature[Disable signature sign and verify, for testing]' \
        '-help[Show help message]' \
        '-no-password[Use empty password for master key]' \
        '-password=-[Master key]:name: ' \
        '-log-level=-[Console log level]:log_level:(trace debug info warning error fatal panic)' \
        '*:arguments:(config public)'
      ;;
    (console)
      _arguments '-config=-[Config file for CovenantSQL, default "~/.cql/config.yaml]:CONF file:_files' \
        '-bypass-signature[Disable signature sign and verify, for testing]' \
        '-help[Show help message]' \
        '-no-password[Use empty password for master key]' \
        '-password=-[Master key]:name: ' \
        '-log-level=-[Console log level]:log_level:(trace debug info warning error fatal panic)' \
        '-adapter=-[Address to serve a database chain adapter]:adapter_addr: ' \
        '-command=-[Run only single command and exit]:cmd: ' \
        '-dsn=-[Database url]:dsn:_cql_dsn' \
        '-explorer=-[Address serve a database chain explorer]:explorer_addr: ' \
        '-file=-[Execute commands from file and exit]:_files' \
        '-no-rc[Do not read start up file]' \
        '-out=-[Record stdout to file]:out_file:_files' \
        '-single-transaction[Execute as a single transaction (if non-interactive)]' \
        '-variable=-[Set variable]:var: '
      ;;
    (create)
      _arguments '-config=-[Config file for CovenantSQL, default "~/.cql/config.yaml]:CONF file:_files' \
        '-bypass-signature[Disable signature sign and verify, for testing]' \
        '-help[Show help message]' \
        '-no-password[Use empty password for master key]' \
        '-password=-[Master key]:name: ' \
        '-log-level=-[Console log level]:log_level:(trace debug info warning error fatal panic)' \
        '-wait-tx-confirm[Wait for transaction confirmation]'
      ;;
    (drop)
      _arguments -C '-config=-[Config file for CovenantSQL, default "~/.cql/config.yaml]:CONF file:_files' \
        '-bypass-signature[Disable signature sign and verify, for testing]' \
        '-help[Show help message]' \
        '-no-password[Use empty password for master key]' \
        '-password=-[Master key]:name: ' \
        '-log-level=-[Console log level]:log_level:(trace debug info warning error fatal panic)' \
        '-wait-tx-confirm[Wait for transaction confirmation]' \
        '1:Database ID:_cql_dsn '
      ;;
    (wallet)
      _arguments '-config=-[Config file for CovenantSQL, default "~/.cql/config.yaml]:CONF file:_files' \
        '-bypass-signature[Disable signature sign and verify, for testing]' \
        '-help[Show help message]' \
        '-no-password[Use empty password for master key]' \
        '-password=-[Master key]:name: ' \
        '-log-level=-[Console log level]:log_level:(trace debug info warning error fatal panic)' \
        '-balance=-[Get specific token balance of current account]:token_type:(Particle Wave All)'
      ;;
    (transfer)
      _arguments '-config=-[Config file for CovenantSQL, default "~/.cql/config.yaml]:CONF file:_files' \
        '-bypass-signature[Disable signature sign and verify, for testing]' \
        '-help[Show help message]' \
        '-no-password[Use empty password for master key]' \
        '-password=-[Master key]:name: ' \
        '-log-level=-[Console log level]:log_level:(trace debug info warning error fatal panic)' \
        '-wait-tx-confirm[Wait for transaction confirmation]'
      ;;
    (grant)
      _arguments '-config=-[Config file for CovenantSQL, default "~/.cql/config.yaml]:CONF file:_files' \
        '-bypass-signature[Disable signature sign and verify, for testing]' \
        '-help[Show help message]' \
        '-no-password[Use empty password for master key]' \
        '-password=-[Master key]:name: ' \
        '-log-level=-[Console log level]:log_level:(trace debug info warning error fatal panic)' \
        '-wait-tx-confirm[Wait for transaction confirmation]'
      ;;
    (mirror)
      _arguments '-config=-[Config file for CovenantSQL, default "~/.cql/config.yaml]:CONF file:_files' \
        '-bypass-signature[Disable signature sign and verify, for testing]' \
        '-help[Show help message]' \
        '-no-password[Use empty password for master key]' \
        '-password=-[Master key]:name: ' \
        '-log-level=-[Console log level]:log_level:(trace debug info warning error fatal panic)' \
        '-bg-log-level=-[Background log level]:bg_log_level:(trace debug info warning error fatal panic)' \
        '-tmp-path=-[Background service temp file path, use os.TempDir for default]:tmp_path:_files' \
        '1:Database ID:_cql_dsn ' \
        '2:Listen Addr:'
      ;;
    (explorer)
      _arguments '-config=-[Config file for CovenantSQL, default "~/.cql/config.yaml]:CONF file:_files' \
        '-bypass-signature[Disable signature sign and verify, for testing]' \
        '-help[Show help message]' \
        '-no-password[Use empty password for master key]' \
        '-password=-[Master key]:name: ' \
        '-log-level=-[Console log level]:log_level:(trace debug info warning error fatal panic)' \
        '-bg-log-level=-[Background log level]:bg_log_level:(trace debug info warning error fatal panic)' \
        '-tmp-path=-[Background service temp file path, use os.TempDir for default]:tmp_path:_files' \
        '1:Listen Addr:'
      ;;
    (adapter)
      _arguments '-config=-[Config file for CovenantSQL, default "~/.cql/config.yaml]:CONF file:_files' \
        '-bypass-signature[Disable signature sign and verify, for testing]' \
        '-help[Show help message]' \
        '-no-password[Use empty password for master key]' \
        '-password=-[Master key]:name: ' \
        '-log-level=-[Console log level]:log_level:(trace debug info warning error fatal panic)' \
        '-bg-log-level=-[Background log level]:bg_log_level:(trace debug info warning error fatal panic)' \
        '-tmp-path=-[Background service temp file path, use os.TempDir for default]:tmp_path:_files' \
        '-mirror=-[Mirror server for adapter to query]:' \
        '1:Listen Addr:'
      ;;
    (idminer)
      _arguments '-config=-[Config file for CovenantSQL, default "~/.cql/config.yaml]:CONF file:_files' \
        '-bypass-signature[Disable signature sign and verify, for testing]' \
        '-help[Show help message]' \
        '-no-password[Use empty password for master key]' \
        '-password=-[Master key]:name: ' \
        '-log-level=-[Console log level]:log_level:(trace debug info warning error fatal panic)' \
        '-difficulty=-[Difficulty for miner to mine nodes and generating nonce (default 24)]:difficulty: ' \
        '-loop[Keep mining until interrupted]'
      ;;
    (rpc)
      _arguments '-config=-[Config file for CovenantSQL, default "~/.cql/config.yaml]:CONF file:_files' \
        '-bypass-signature[Disable signature sign and verify, for testing]' \
        '-help[Show help message]' \
        '-no-password[Use empty password for master key]' \
        '-password=-[Master key]:name: ' \
        '-log-level=-[Console log level]:log_level:(trace debug info warning error fatal panic)' \
        '-wait-tx-confirm[Wait for transaction confirmation]' \
        '-endpoint=-[RPC endpoint Node ID to do test call]:' \
        '-name=-[RPC name to do test call]:' \
        '-req=-[RPC request to do test call, in json format]:'
      ;;
  esac
}

_cql() {
  local -a commands

  commands=(
	"generate:generate config related file or keys"
	"console:run a console for interactive sql operation"
	"create:create a database"
	"drop:drop a database by dsn or database id"
	"wallet:get the wallet address and the balance of current account"
	"transfer:transfer token to target account"
	"grant:grant a user's permissions on specific sqlchain"
	"mirror:start a SQLChain database mirror"
	"explorer:start a SQLChain explorer explorer"
	"adapter:start a SQLChain adapter"
	"idminer:calculate nonce and node id for config.yaml file"
	"rpc:make a rpc request"
	"version:show build version infomation"
	"help:show help for sub command"
  )

  _arguments -C \
    '1:cmd:->cmds' \
    '*:: :->args' \

  case "$state" in
    (cmds)
      _describe -t commands 'commands' commands
      ;;
    (*)
      _cql_args
      ;;
  esac
}

_cql "$@"

