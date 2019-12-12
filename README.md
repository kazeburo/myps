# myps

Like pgrep and pkill, grep MySQL processlist and kill threads.

## Example

grep by TIME

```
$ myps grep --duration 0
ID:142  USER:root       HOST:localhost:59393    DB:     COMMAND:Query   TIME:57 STATE:User sleep        INFO:select  sleep(3600)
ID:150  USER:root       HOST:localhost:59814    DB:     COMMAND:Sleep   TIME:2  STATE:  INFO:
ID:145  USER:root       HOST:localhost:59800    DB:     COMMAND:Query   TIME:13 STATE:User sleep        INFO:select sleep(360)
```

grep by TIME and Query

```
$ myps grep --duration 10 --info 'select%'
ID:142  USER:root       HOST:localhost:59393    DB:     COMMAND:Query   TIME:86 STATE:User sleep        INFO:select  sleep(3600)
ID:145  USER:root       HOST:localhost:59800    DB:     COMMAND:Query   TIME:42 STATE:User sleep        INFO:select sleep(360)
```

and kill them

```
$ myps kill --duration 10 --info 'select%'
KILLED:142      USER:root       HOST:localhost:59393    DB:     COMMAND:Query   TIME:129        STATE:User sleep        INFO:select  sleep(3600)
KILLED:145      USER:root       HOST:localhost:59800    DB:     COMMAND:Query   TIME:85 STATE:User sleep        INFO:select sleep(360)
```

By default, myps use `$HOME/.my.cnf` if exists.

## Usage

```
$ ./myps -h grep
Usage:
  myps [OPTIONS] <grep | kill>

Help Options:
  -h, --help  Show this help message

Available commands:
  grep
  kill
```

### grep

```
% ./myps grep -h
Usage:
  myps [OPTIONS] grep [grep-OPTIONS]

Help Options:
  -h, --help                Show this help message

[grep command options]
          --mysql-host=     Hostname
          --mysql-port=     Port
          --mysql-user=     Username
          --mysql-password= Password
          --mysql-socket=   path to mysql listen sock
          --defaults-file=  path to defaults-file
          --timeout=        Timeout to connect mysql (default: 5s)
          --duration=       display/kill process only time >= duration
          --user=           display/kill process of user name
          --db=             display/kill process of db name. % wildcard allowed
          --command=        display/kill process of command. % wildcard allowed
          --state=          display/kill process of state. % wildcard allowed
          --info=           display/kill process of info(query). % wildcard allowed
```

### kill

```
% ./myps kill -h
Usage:
  myps [OPTIONS] kill [kill-OPTIONS]

Help Options:
  -h, --help                Show this help message

[kill command options]
          --mysql-host=     Hostname
          --mysql-port=     Port
          --mysql-user=     Username
          --mysql-password= Password
          --mysql-socket=   path to mysql listen sock
          --defaults-file=  path to defaults-file
          --timeout=        Timeout to connect mysql (default: 5s)
          --duration=       display/kill process only time >= duration
          --user=           display/kill process of user name
          --db=             display/kill process of db name. % wildcard allowed
          --command=        display/kill process of command. % wildcard allowed
          --state=          display/kill process of state. % wildcard allowed
          --info=           display/kill process of info(query). % wildcard allowed
```
