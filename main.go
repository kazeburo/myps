package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/andrew-d/go-termutil"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jessevdk/go-flags"
	"github.com/mgutz/ansi"
	"github.com/percona/go-mysql/dsn"
	"github.com/vaughan0/go-ini"
)

type commonSetting struct {
	MySQLHost              string        `long:"mysql-host" description:"Hostname"`
	MySQLPort              string        `long:"mysql-port" description:"Port"`
	MySQLUser              string        `long:"mysql-user" description:"Username"`
	MySQLPass              *string       `long:"mysql-password" description:"Password"`
	MySQLSocket            string        `long:"mysql-socket" description:"path to mysql listen sock"`
	MySQLTimeout           time.Duration `long:"mysql-timeout" default:"30s" description:"Timeout to connect mysql"`
	MySQLDefaultsExtraFile string        `long:"defaults-extra-file" description:"path to defaults-extra-file"`
}

type filterSetting struct {
	Time    *string `group:"filter" short:"t" long:"time" description:"display/kill process only >= time"`
	User    *string `group:"filter" short:"u" long:"user" description:"display/kill process of user name"`
	DB      *string `group:"filter" short:"d" long:"db" description:"display/kill process of db name. % wildcard allowed"`
	Command *string `group:"filter" short:"c" long:"command" description:"display/kill process of command. % wildcard allowed"`
	State   *string `group:"filter" short:"s" long:"state" description:"display/kill process of state. % wildcard allowed"`
	Info    *string `group:"filter" short:"i" long:"info" description:"display/kill process of info(query). % wildcard allowed"`
}

type displaySetting struct {
	Debug bool `group:"display" short:"D" long:"debug" description:"Display debug"`
	Full  bool `group:"display" short:"f" long:"full" description:"Display query all (like show full processlist)"`
}

type processInfo struct {
	ID      int64  `json:"id"`
	USER    string `json:"user"`
	HOST    string `json:"host"`
	DB      string `json:"db"`
	COMMAND string `json:"command"`
	TIME    int64  `json:"time"`
	STATE   string `json:"state"`
	INFO    string `json:"info"`
}

type grepOpts struct {
	commonSetting
	filterSetting
	displaySetting
}

type killOpts struct {
	commonSetting
	filterSetting
	displaySetting
	YesKill bool `group:"kill" short:"y" long:"yes" description:"Skip confirmation prompt to kill threads"`
}

type mainOpts struct {
	GrepCmd grepOpts `command:"grep"`
	KillCmd killOpts `command:"kill"`
}

func openDB(opts commonSetting, debug bool) (*sql.DB, error) {
	dsn, err := dsn.Defaults("")
	if err != nil {
		return nil, err
	}

	if opts.MySQLDefaultsExtraFile != "" {
		i, err := ini.LoadFile(opts.MySQLDefaultsExtraFile)
		if err != nil {
			return nil, err
		}
		section := i.Section("client")
		user, ok := section["user"]
		if ok {
			dsn.Username = user
		}
		password, ok := section["password"]
		if ok {
			dsn.Password = password
		}
		socket, ok := section["socket"]
		if ok {
			dsn.Socket = socket
		}
		host, ok := section["host"]
		if ok {
			dsn.Hostname = host
		}
		port, ok := section["port"]
		if ok {
			dsn.Port = port
		}
	}
	if opts.MySQLHost != "" {
		dsn.Hostname = opts.MySQLHost
	}
	if opts.MySQLPort != "" {
		dsn.Port = opts.MySQLPort
	}
	if opts.MySQLUser != "" {
		dsn.Username = opts.MySQLUser
	}
	if opts.MySQLPass != nil {
		dsn.Password = *opts.MySQLPass
	}
	if opts.MySQLSocket != "" {
		dsn.Socket = opts.MySQLSocket
	}

	if dsn.Username == "" {
		dsn.Username = os.Getenv("USER")
	}

	dsn.Params = append(dsn.Params, "interpolateParams=true")
	dsn.Params = append(dsn.Params, fmt.Sprintf("timeout=%s", opts.MySQLTimeout.String()))
	dsnString := dsn.String()
	if debug {
		dsn.Password = "xxxx"
		log.Printf("DSN: %s", dsn.String())
	}
	db, err := sql.Open("mysql", dsnString)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func checkCriteria(opts *filterSetting, args []string, command string) error {
	if opts.Info == nil && len(args) > 0 {
		opts.Info = &args[0]
	}
	if opts.Time == nil &&
		opts.User == nil &&
		opts.DB == nil &&
		opts.Command == nil &&
		opts.State == nil &&
		opts.Info == nil {
		return fmt.Errorf("no matching criteria specified.\n try `%s %s --help' for more information", os.Args[0], command)
	}
	return nil
}

func processList(conn *sql.Conn, opts filterSetting, debug bool) ([]processInfo, error) {
	args := []interface{}{}
	where := []string{}
	processList := []processInfo{}

	if opts.Time != nil {
		args = append(args, *opts.Time)
		where = append(where, "TIME >= ?")
	}

	if opts.User != nil {
		args = append(args, *opts.User)
		where = append(where, `IFNULL(USER,"") LIKE ?`)
	}
	if opts.DB != nil {
		args = append(args, *opts.DB)
		where = append(where, `IFNULL(DB,"") LIKE ?`)
	}
	if opts.Command != nil {
		args = append(args, *opts.Command)
		where = append(where, `IFNULL(COMMAND,"") LIKE ?`)
	}
	if opts.State != nil {
		args = append(args, *opts.State)
		where = append(where, `IFNULL(STATE,"") LIKE ?`)
	}
	if opts.Info != nil {
		args = append(args, *opts.Info)
		where = append(where, `IFNULL(INFO,"") LIKE ?`)
	}

	query := `SELECT /* SHOW PROCESSLIST */ ID, IFNULL(USER,"") USER, IFNULL(HOST,"") HOST, IFNULL(DB,"") DB, IFNULL(COMMAND,"") COMMAND, TIME, IFNULL(STATE,"") STATE, IFNULL(INFO,"") INFO FROM information_schema.PROCESSLIST WHERE ID != CONNECTION_ID() AND `
	query = query + strings.Join(where, " AND ")
	if debug {
		log.Printf("Query: %s", query)
		log.Printf("Args: %s", args)
	}
	rows, err := conn.QueryContext(context.Background(), query, args...)
	if err != nil {
		return processList, err
	}
	defer rows.Close()
	for rows.Next() {
		p := processInfo{}
		err := rows.Scan(&p.ID, &p.USER, &p.HOST, &p.DB, &p.COMMAND, &p.TIME, &p.STATE, &p.INFO)
		if err != nil {
			panic(err)
		}
		processList = append(processList, p)
	}
	return processList, err
}

var infoLabelColor = "green"
var warnLabelColor = "red"
var valueColor = "magenta"

func makeField(label, value, labelColor string) string {
	v := fmt.Sprintf("%q", value)
	v = strings.TrimPrefix(v, `"`)
	v = strings.TrimSuffix(v, `"`)
	v = strings.ReplaceAll(v, `\"`, `"`)
	if termutil.Isatty(os.Stdout.Fd()) {
		return ansi.Color(label, labelColor) + ":" + ansi.Color(v, valueColor)
	}
	return label + ":" + v
}

var maxDefaultInfoLength = 110

func makeLTSVln(pi processInfo, full bool, idLabel string) string {
	buf := []string{}
	if idLabel == "ID" {
		buf = append(buf, makeField(idLabel, fmt.Sprintf("%d", pi.ID), infoLabelColor))
	} else {
		buf = append(buf, makeField(idLabel, fmt.Sprintf("%d", pi.ID), warnLabelColor))
	}
	buf = append(buf, makeField("USER", pi.USER, infoLabelColor))
	buf = append(buf, makeField("HOST", pi.HOST, infoLabelColor))
	buf = append(buf, makeField("DB", pi.DB, infoLabelColor))
	buf = append(buf, makeField("COMMAND", pi.COMMAND, infoLabelColor))
	buf = append(buf, makeField("TIME", fmt.Sprintf("%d", pi.TIME), infoLabelColor))
	buf = append(buf, makeField("STATE", pi.STATE, infoLabelColor))
	sub := []rune(pi.INFO)
	if full || len(sub) < maxDefaultInfoLength {
		buf = append(buf, makeField("INFO", pi.INFO, infoLabelColor))
	} else {
		buf = append(buf, makeField("INFO", string(sub[:maxDefaultInfoLength]), infoLabelColor))
	}

	return strings.Join(buf, "\t") + "\n"
}

var notFound = false

func (opts *grepOpts) Execute(args []string) error {
	err := checkCriteria(&opts.filterSetting, args, "grep")
	if err != nil {
		return err
	}
	db, err := openDB(opts.commonSetting, opts.Debug)
	if err != nil {
		return err
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		return err
	}
	defer conn.Close()
	pl, err := processList(conn, opts.filterSetting, opts.Debug)
	if err != nil {
		return err
	}
	if len(pl) == 0 {
		notFound = true
		return nil
	}
	for _, pi := range pl {
		os.Stdout.WriteString(makeLTSVln(pi, opts.Full, "ID"))
	}

	return nil
}

var confirmReg = regexp.MustCompile("^[yY]$")

func (opts *killOpts) Execute(args []string) error {
	err := checkCriteria(&opts.filterSetting, args, "kill")
	if err != nil {
		return err
	}
	if !opts.YesKill {
		confirmMsg := "Are you sure you want to kill threads? [y/N]:"
		if termutil.Isatty(os.Stdout.Fd()) {
			confirmMsg = ansi.Color(confirmMsg, "magenta")

		}
		fmt.Print(confirmMsg)
		reader := bufio.NewReader(os.Stdin)
		confirm, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		confirm = strings.TrimRight(confirm, "\n")
		if !confirmReg.MatchString(confirm) {
			return nil
		}
	}
	db, err := openDB(opts.commonSetting, opts.Debug)
	if err != nil {
		return err
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		return err
	}
	defer conn.Close()
	pl, err := processList(conn, opts.filterSetting, opts.Debug)
	if err != nil {
		return err
	}
	if len(pl) == 0 {
		// no set notFound = true for kill
		return nil
	}
	for _, pi := range pl {
		if opts.Debug {
			log.Printf("Query: KILL %d", pi.ID)
		}
		_, err := conn.ExecContext(context.Background(), "KILL ?", pi.ID)
		if err != nil {
			if mysqlErr, ok := err.(*mysql.MySQLError); ok {
				// Error 1094: Unknown thread id: 300
				if mysqlErr.Number != 1094 {
					return err
				}
			} else {
				return err
			}
		}
		os.Stdout.WriteString(makeLTSVln(pi, opts.Full, "KILLED"))
	}

	return nil
}

func main() {
	opts := mainOpts{}
	psr := flags.NewParser(&opts, flags.Default)
	_, err := psr.Parse()
	if err != nil || notFound {
		os.Exit(1)
	}
}
