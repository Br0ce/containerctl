# sshconfig

[![ci](https://github.com/Br0ce/sshconfig/actions/workflows/ci.yml/badge.svg)](https://github.com/Br0ce/sshconfig/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Br0ce/sshconfig)](go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/Br0ce/sshconfig.svg)](https://pkg.go.dev/github.com/Br0ce/sshconfig)

Your `~/.ssh/config`, resolved in Go the way OpenSSH does it.

`sshconfig` reads OpenSSH client config files and tells you which settings apply to a host, much like `ssh -G`. It only parses: it doesn't open connections and doesn't modify config files. Use it to configure an SSH client of your own, e.g. one built on [x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh), from the user's existing config.

```sh
go get github.com/Br0ce/sshconfig
```

## Usage

Given this `~/.ssh/config`:

```sshconfig
Host web
    HostName web.example.com
    User deploy
    Port 2222
    IdentityFile /home/max/.ssh/id_web

Host cmp*
    HostName %h.cluster.example.com

Host *
    IdentityFile ~/.ssh/id_%r_%h
```

`web` spells out every value. `cmp*` covers a whole set of hosts, `cmp1`, `cmp2` and so on, and `Host *` gives every host a default key. Both leave the details to `~` and `%`-tokens: `%h` stands for the host name, `%r` for the remote user.

Load the files `ssh` itself reads and look up a host. A file that doesn't exist is treated as empty. `Get` returns the value in effect, which for `web` is simply what the config says:

```go
user, err := sshconfig.UserConfig()     // ~/.ssh/config
...
system, err := sshconfig.SystemConfig() // /etc/ssh/ssh_config
...
web := sshconfig.Lookup("web",
	sshconfig.WithUserConfig(user),     // read first, its values win
	sshconfig.WithSystemConfig(system), // fills in what the user config leaves unset
)
web.Get("HostName")     // "web.example.com"
web.Get("User")         // "deploy"
web.Get("Port")         // "2222"
web.Get("IdentityFile") // "/home/max/.ssh/id_web", the first value wins over Host *
```

For `cmp1`, `Get` resolves `~` and the tokens the way `ssh` would. Some values come from the config, others from the user and host running the program, as for `ssh` itself:

```go
cmp := sshconfig.Lookup("cmp1",
	sshconfig.WithUserConfig(user),
	sshconfig.WithSystemConfig(system),
)
cmp.Get("HostName")     // "cmp1.cluster.example.com"
                        //   %h -> the alias looked up
cmp.Get("IdentityFile") // "/home/max/.ssh/id_max_cmp1.cluster.example.com"
                        //   ~  -> home of the local user, e.g. max
                        //   %r -> local user, e.g. max
                        //   %h -> HostName, expanded first
cmp.GetRaw("HostName")  // "%h.cluster.example.com", as written in the config
```

`Get` returns an error if a value uses a token its keyword doesn't accept according to ssh_config(5), or needs a local value which is not provided. To resolve a config for another user, or with fixed values in tests, pass the local side with `WithLocalAs`. For keywords that may be given more than once, `GetAll` and `GetAllRaw` return every value in order.

To read a config from any other path, like `ssh -F`, use `DecodeFile`. Unlike `UserConfig`, it returns an error if the file doesn't exist. Relative `Include` paths resolve against the file's directory, while `ssh -F` uses `~/.ssh`; pass `WithIncludeDir` to change that.

```go
cfg, err := sshconfig.DecodeFile("deploy/ssh_config")
...
vals := sshconfig.Lookup("web", sshconfig.WithUserConfig(cfg))
```

## How it resolves

The rules follow [ssh_config(5)](https://man.openbsd.org/ssh_config):

- The first value found for a keyword wins, so general `Host *` settings go last.
- `Host` patterns match the alias you look up; `Match host` matches its `HostName`.
- `Include` is read where it appears, and only if its block applies.
- `~` and `%`-tokens are expanded per keyword, each with its own set of allowed tokens.

## Limitations

- `Match` supports `all`, `host` and `originalhost`; blocks with other criteria never match.
- Tokens only known while connecting, such as `%T`, are not supported.
- Arguments are split and unquoted like those of other keywords, so command keywords such as `ProxyCommand` or `RemoteCommand` lose their quoting. `ProxyJump` is not affected.
- Keywords with several paths, such as `UserKnownHostsFile`, return them separated by spaces, so a path containing a space can't be told apart.
- No defaults for unset keywords, no hostname canonicalization, no permission checks.

## License

MIT, see [LICENSE](LICENSE).
