// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package datastore

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/golang/glog"
	sqlite3 "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type sqliteDB struct {
	db            *sql.DB
	tdb           *sql.DB
	dbName        string
	tdbName       string
	tables        []persistentData
	tableInitPath string
	workloadsPath string
	dbLock        *sync.Mutex
	tdbLock       *sync.RWMutex
}

type persistentData interface {
	Init() error
	Populate() error
	Create(...string) error
	Name() string
	DB() *sql.DB
}

type namedData struct {
	ds   *sqliteDB
	name string
	db   *sql.DB
}

func (d namedData) Create(record ...string) (err error) {
	err = d.ds.create(d.name, record)
	return
}

func (d namedData) Populate() (err error) {
	return nil
}

func (d namedData) Name() (name string) {
	return d.name
}

func (d namedData) DB() *sql.DB {
	return d.db
}

func (d namedData) ReadCsv() (records [][]string, err error) {
	f, err := os.Open(fmt.Sprintf("%s/%s.csv", d.ds.tableInitPath, d.name))
	if err != nil {
		return
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	r.Comment = '#'

	records, err = r.ReadAll()
	if err != nil {
		return
	}
	return
}

type logData struct {
	namedData
}

func (d logData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS log
		(
		id integer primary key,
		tenant_id varchar(32),
		type string,
		message string,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
		);`
	err = d.ds.exec(d.db, cmd)
	return
}

type subnetData struct {
	namedData
}

func (d subnetData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS tenant_network
		(
		tenant_id varchar(32),
		subnet int,
		rest int,
		foreign key(tenant_id) references tenants(id)
		);`
	err = d.ds.exec(d.db, cmd)
	return
}

// Handling of Limit specific Data
type limitsData struct {
	namedData
}

func (d limitsData) Populate() (err error) {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		resourceID, _ := strconv.Atoi(line[0])
		tenantID := line[1]
		maxValue, _ := strconv.Atoi(line[2])
		err = d.ds.create(d.name, resourceID, tenantID, maxValue)
		if err != nil {
			glog.V(2).Info("could not add limit: ", err)
		}
	}
	return
}

func (d limitsData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS limits
		(
		resource_id integer,
		tenant_id varchar(32),
		max_value integer,
		foreign key(resource_id) references resources(id),
		foreign key(tenant_id) references tenants(id)
		);`
	err = d.ds.exec(d.db, cmd)
	return
}

// Handling of Instance specific data
type instanceData struct {
	namedData
}

func (d instanceData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS instances
		(
		id string primary key,
		tenant_id string,
		workload_id string,
		mac_address string,
		ip string,
		foreign key(tenant_id) references tenants(id),
		foreign key(workload_id) references workload_template(id),
		unique(tenant_id, ip, mac_address)
		);`
	err = d.ds.exec(d.db, cmd)
	return
}

// Resources data
type resourceData struct {
	namedData
}

func (d resourceData) Populate() (err error) {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		id, _ := strconv.Atoi(line[0])
		name := line[1]
		err = d.ds.create(d.name, id, name)
		if err != nil {
			glog.V(2).Info("could not add resource: ", err)
		}
	}
	return
}

func (d resourceData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS resources
		(
		id int primary key,
		name text
		);`
	err = d.ds.exec(d.db, cmd)
	return
}

// Tenants data
type tenantData struct {
	namedData
}

func (d tenantData) Populate() (err error) {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		id := line[0]
		name := line[1]
		mac := line[2]
		if err != nil {
			glog.V(2).Info("could not add tenant: ", err)
		}
		err = d.ds.create(d.name, id, name, "", mac, "")
		if err != nil {
			glog.V(2).Info("could not add tenant: ", err)
		}
	}
	return
}

func (d tenantData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS tenants
		(
		id varchar(32) primary key,
		name text,
		cnci_id varchar(32) default null,
		cnci_mac string default null,
		cnci_ip string default null
		);`
	err = d.ds.exec(d.db, cmd)
	return
}

// usage data
type usageData struct {
	namedData
}

func (d usageData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS usage
		(
		instance_id string,
		resource_id int,
		value int,
		foreign key(instance_id) references instances(id),
		foreign key(resource_id) references resources(id)
		);
		CREATE UNIQUE INDEX IF NOT EXISTS myindex
		ON usage(instance_id, resource_id);`
	err = d.ds.exec(d.db, cmd)
	return
}

// workload resources
type workloadResourceData struct {
	namedData
}

func (d workloadResourceData) Populate() (err error) {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		workloadID := line[0]
		resourceID, _ := strconv.Atoi(line[1])
		defaultValue, _ := strconv.Atoi(line[2])
		estimatedValue, _ := strconv.Atoi(line[3])
		mandatory, _ := strconv.Atoi(line[4])
		err = d.ds.create(d.name, workloadID, resourceID, defaultValue, estimatedValue, mandatory)
		if err != nil {
			glog.V(2).Info("could not add workload: ", err)
		}
	}
	return
}

func (d workloadResourceData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS workload_resources
		(
		workload_id varchar(32),
		resource_id int,
		default_value int,
		estimated_value int,
		mandatory int,
		foreign key(workload_id) references workload_template(id),
		foreign key(resource_id) references resources(id)
		);
		CREATE UNIQUE INDEX IF NOT EXISTS wlr_index
		ON workload_resources(workload_id, resource_id);`
	err = d.ds.exec(d.db, cmd)
	return
}

// workload template data
type workloadTemplateData struct {
	namedData
}

func (d workloadTemplateData) Populate() (err error) {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		id := line[0]
		description := line[1]
		filename := line[2]
		fwType := line[3]
		vmType := line[4]
		imageID := line[5]
		imageName := line[6]
		internal := line[7]
		err = d.ds.create(d.name, id, description, filename, fwType, vmType, imageID, imageName, internal)
		if err != nil {
			glog.V(2).Info("could not add workload: ", err)
		}
	}
	return
}

func (d workloadTemplateData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS workload_template
		(
		id varchar(32) primary key,
		description text,
		filename text,
		fw_type text,
		vm_type text,
		image_id varchar(32),
		image_name text,
		internal integer
		);`
	err = d.ds.exec(d.db, cmd)
	return
}

// statistics
type nodeStatisticsData struct {
	namedData
}

func (d nodeStatisticsData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS node_statistics
		(
			id integer primary key autoincrement not null,
			node_id varchar(32),
			mem_total_mb int,
			mem_available_mb int,
			disk_total_mb int,
			disk_available_mb int,
			load int,
			cpus_online int,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
		);`
	err = d.ds.exec(d.db, cmd)
	return
}

type instanceStatisticsData struct {
	namedData
}

func (d instanceStatisticsData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS instance_statistics
		(
			id integer primary key autoincrement not null,
			instance_id varchar(32),
			memory_usage_mb int,
			disk_usage_mb int,
			cpu_usage int,
			state string,
			node_id varchar(32),
			ssh_ip string,
			ssh_port int,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
		);`
	err = d.ds.exec(d.db, cmd)
	return
}

type frameStatisticsData struct {
	namedData
}

func (d frameStatisticsData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS frame_statistics
		(
			id integer primary key autoincrement not null,
			label string,
			type string,
			operand string,
			start_timestamp DATETIME,
			end_timestamp DATETIME
		);`
	err = d.ds.exec(d.db, cmd)
	return
}

type traceData struct {
	namedData
}

func (d traceData) Init() (err error) {
	cmd := `CREATE TABLE IF NOT EXISTS trace_data
		(
			id integer primary key autoincrement not null,
			frame_id int,
			ssntp_uuid varchar(32),
			tx_timestamp DATETIME,
			rx_timestamp DATETIME,
			foreign key(frame_id) references frame_statistics(id)
		);`
	err = d.ds.exec(d.db, cmd)
	return
}

func (ds *sqliteDB) exec(db *sql.DB, cmd string) (err error) {
	glog.V(2).Info("exec: ", cmd)

	tx, err := db.Begin()
	if err != nil {
		return
	}

	_, err = tx.Exec(cmd)
	if err != nil {
		tx.Rollback()
		return
	}
	tx.Commit()
	return
}

func (ds *sqliteDB) create(tableName string, record ...interface{}) (err error) {
	// get database location of this table
	db := ds.getTableDB(tableName)

	if db == nil {
		err = errors.New("Bad table name")
		return
	}

	var values []string
	for _, val := range record {
		v := reflect.ValueOf(val)
		var newval string
		// enclose strings in quotes to not confuse sqlite
		if v.Kind() == reflect.String {
			newval = fmt.Sprintf("'%v'", val)
		} else {
			newval = fmt.Sprintf("%v", val)
		}
		values = append(values, newval)
	}
	args := strings.Join(values, ",")
	cmd := "INSERT or IGNORE into " + tableName + " VALUES (" + args + ");"
	err = ds.exec(db, cmd)
	return
}

func (ds *sqliteDB) getTableDB(name string) *sql.DB {
	for _, table := range ds.tables {
		n := table.Name()
		if n == name {
			return table.DB()
		}
	}
	return nil
}

// Init initializes the private data for the database object.
// The sql tables are populated with initial data from csv
// files if this is the first time the database has been
// created.  The datastore caches are also filled.
func getPersistentStore(config Config) (ps persistentStore, err error) {
	var ds = &sqliteDB{}

	err = ds.Connect(config.PersistentURI, config.TransientURI)
	if err != nil {
		return
	}

	ds.dbLock = &sync.Mutex{}
	ds.tdbLock = &sync.RWMutex{}

	ds.tables = []persistentData{
		resourceData{namedData{ds: ds, name: "resources", db: ds.db}},
		tenantData{namedData{ds: ds, name: "tenants", db: ds.db}},
		limitsData{namedData{ds: ds, name: "limits", db: ds.db}},
		instanceData{namedData{ds: ds, name: "instances", db: ds.db}},
		workloadTemplateData{namedData{ds: ds, name: "workload_template", db: ds.db}},
		workloadResourceData{namedData{ds: ds, name: "workload_resources", db: ds.db}},
		usageData{namedData{ds: ds, name: "usage", db: ds.db}},
		nodeStatisticsData{namedData{ds: ds, name: "node_statistics", db: ds.tdb}},
		logData{namedData{ds: ds, name: "log", db: ds.tdb}},
		subnetData{namedData{ds: ds, name: "tenant_network", db: ds.db}},
		instanceStatisticsData{namedData{ds: ds, name: "instance_statistics", db: ds.tdb}},
		frameStatisticsData{namedData{ds: ds, name: "frame_statistics", db: ds.tdb}},
		traceData{namedData{ds: ds, name: "trace_data", db: ds.tdb}},
	}

	ds.tableInitPath = config.InitTablesPath
	ds.workloadsPath = config.InitWorkloadsPath

	for _, table := range ds.tables {
		err = table.Init()
		if err != nil {
			return
		}
	}

	for _, table := range ds.tables {
		// Populate failures are not fatal, because it could just mean
		// there's no initial data to populate
		_ = table.Populate()
	}

	return ds, nil
}

// Connect creates two sqlite3 databases.  One database is for
// persistent state that needs to be restored on restart, the
// other is for transient data that does not need to be restored
// on restart.
func (ds *sqliteDB) Connect(persistentURI string, transientURI string) (err error) {
	sql.Register("sqlite_attach_tdb", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			cmd := fmt.Sprintf("ATTACH '%s' AS tdb", transientURI)
			conn.Exec(cmd, nil)
			return nil
		},
	})

	connectString := persistentURI
	datastore, err := sql.Open("sqlite_attach_tdb", connectString)
	if err != nil {
		return err
	}

	ds.dbName = persistentURI
	ds.tdbName = transientURI

	_, err = datastore.Exec("PRAGMA page_size = 32768;")
	if err != nil {
		glog.Warning("unable to increase page size size", err)
	}

	_, err = datastore.Exec("PRAGMA synchronous = OFF")
	if err != nil {
		glog.Warning("unable to turn off synchronous", err)
	}

	_, err = datastore.Exec("PRAGMA temp_store = MEMORY")
	if err != nil {
		glog.Warning("unable to change temp_store", err)
	}

	err = datastore.Ping()
	if err != nil {
		glog.Warning("unable to ping database")
		return
	}

	ds.db = datastore

	// if I turn off foreign key support, I can do some work
	// asynchronously
	//_, err = datastore.Exec("PRAGMA foreign_keys = ON")
	//if err != nil {
	//	glog.Warning("unable to turn on foreign key support", err)
	//}

	// TBD - what's the best busy_timeout value (ms)?
	_, err = datastore.Exec("PRAGMA busy_timeout = 1000")
	if err != nil {
		glog.Warning("unable to set busy_timeout", err)
	}

	_, err = datastore.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		glog.Warning("unable to set journal_mode", err)
	}

	connectString = transientURI
	sql.Register("sqlite_attach_db", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			cmd := fmt.Sprintf("ATTACH '%s' AS db", persistentURI)
			conn.Exec(cmd, nil)
			return nil
		},
	})
	datastore, err = sql.Open("sqlite_attach_db", connectString)
	if err != nil {
		return err
	}

	_, err = datastore.Exec("PRAGMA page_size = 32768;")
	if err != nil {
		glog.Warning("unable to increase page size size", err)
	}

	err = datastore.Ping()
	if err != nil {
		glog.Warning("unable to ping database")
		return
	}

	ds.tdb = datastore

	_, err = datastore.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		glog.Warning("unable to turn on foreign key support", err)
	}

	// TBD - what's the best busy_timeout value (ms)?
	_, err = datastore.Exec("PRAGMA busy_timeout = 500")
	if err != nil {
		glog.Warning("unable to set busy_timeout", err)
	}

	_, err = datastore.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		glog.Warning("unable to set journal_mode", err)
	}

	_, err = datastore.Exec("PRAGMA synchronous = OFF")
	if err != nil {
		glog.Warning("unable to turn off synchronous", err)
	}

	_, err = datastore.Exec("PRAGMA temp_store = MEMORY")
	if err != nil {
		glog.Warning("unable to change temp_store", err)
	}

	return
}

// Disconnect is used to close the connection to the sql database
func (ds *sqliteDB) disconnect() {
	ds.db.Close()
}

func (ds *sqliteDB) logEvent(tenantID string, eventType string, message string) error {
	glog.V(2).Info("log event: ", message)
	cmd := `INSERT INTO log (tenant_id, type, message)
		VALUES('%s', '%s', '%s');`

	ds.tdbLock.Lock()

	str := fmt.Sprintf(cmd, tenantID, eventType, message)
	err := ds.exec(ds.getTableDB("log"), str)
	if err != nil {
		glog.V(2).Info("could not log event: ", message, " ", err)
	}

	ds.tdbLock.Unlock()

	return err
}

// ClearLog will remove all the event entries from the event log
func (ds *sqliteDB) clearLog() error {
	db := ds.getTableDB("log")

	ds.tdbLock.Lock()

	err := ds.exec(db, "DELETE FROM log")
	if err != nil {
		glog.V(2).Info("could not clear log: ", err)
	}

	ds.tdbLock.Unlock()

	return err
}

// GetCNCIWorkloadID returns the UUID of the workload template
// for the CNCI workload
func (ds *sqliteDB) getCNCIWorkloadID() (id string, err error) {
	db := ds.getTableDB("workload_template")

	err = db.QueryRow("SELECT id FROM workload_template WHERE description = 'CNCI'").Scan(&id)
	if err != nil {
		return
	}
	return
}

func (ds *sqliteDB) getConfigNoCache(id string) (config string, err error) {
	var configFile string

	db := ds.getTableDB("workload_template")

	err = db.QueryRow("SELECT filename FROM workload_template where id = ?", id).Scan(&configFile)

	if err != nil {
		return config, err
	}

	path := fmt.Sprintf("%s/%s", ds.workloadsPath, configFile)
	bytes, err := ioutil.ReadFile(path)
	config = string(bytes)
	return config, err
}

func (ds *sqliteDB) getWorkloadDefaults(id string) (defaults []payloads.RequestedResource, err error) {
	query := `SELECT resources.name, default_value, mandatory FROM workload_resources
		  JOIN resources
		  ON workload_resources.resource_id=resources.id
		  WHERE workload_id = ?`
	db := ds.getTableDB("workload_resources")

	rows, err := db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var val int
		var rname string
		var mandatory bool

		err = rows.Scan(&rname, &val, &mandatory)
		if err != nil {
			return nil, err
		}
		r := payloads.RequestedResource{
			Type:      payloads.Resource(rname),
			Value:     val,
			Mandatory: mandatory,
		}
		defaults = append(defaults, r)
	}
	return
}

func (ds *sqliteDB) addLimit(tenantID string, resourceID int, limit int) (err error) {
	ds.dbLock.Lock()
	err = ds.create("limits", resourceID, tenantID, limit)
	ds.dbLock.Unlock()
	return
}

func (ds *sqliteDB) getTenantResources(id string) (resources []*types.Resource, err error) {
	query := `WITH instances_usage AS
		 (
			 SELECT resource_id, value
			 FROM usage
			 LEFT JOIN instances
			 ON usage.instance_id = instances.id
			 WHERE instances.tenant_id = ?
		 )
		 SELECT resources.name, resources.id, limits.max_value,
		 CASE resources.id
		 WHEN resources.id = 1 then
		 (
			 SELECT COUNT(instances.id)
			 FROM instances
			 WHERE instances.tenant_id = ?
		 )
		 ELSE SUM(instances_usage.value)
		 END
		 FROM resources
		 LEFT JOIN instances_usage
		 ON instances_usage.resource_id = resources.id
		 LEFT JOIN limits
		 ON resources.id=limits.resource_id
		 AND limits.tenant_id = ?
		 GROUP BY resources.id`
	datastore := ds.db

	rows, err := datastore.Query(query, id, id, id)
	if err != nil {
		glog.Warning("Failed to get tenant usage")
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		var sqlMaxVal sql.NullInt64
		var sqlCurVal sql.NullInt64
		var maxVal = -1
		var curVal = 0

		err = rows.Scan(&name, &id, &sqlMaxVal, &sqlCurVal)
		if err != nil {
			return nil, err
		}

		if sqlMaxVal.Valid {
			maxVal = int(sqlMaxVal.Int64)
		}
		if sqlCurVal.Valid {
			curVal = int(sqlCurVal.Int64)
		}
		r := types.Resource{
			Rname: name,
			Rtype: id,
			Limit: maxVal,
			Usage: curVal,
		}
		resources = append(resources, &r)
	}

	return
}

func (ds *sqliteDB) addTenant(id string, MAC string) (err error) {
	ds.dbLock.Lock()
	err = ds.create("tenants", id, "", "", MAC, "")
	ds.dbLock.Unlock()
	return
}

func (ds *sqliteDB) getTenantNoCache(id string) (t *tenant, err error) {
	query := `SELECT	tenants.id,
				tenants.name,
				tenants.cnci_id,
				tenants.cnci_mac,
				tenants.cnci_ip
		  FROM tenants
		  WHERE tenants.id = ?`
	datastore := ds.db

	row := datastore.QueryRow(query, id)

	t = new(tenant)

	err = row.Scan(&t.Id, &t.Name, &t.CNCIID, &t.CNCIMAC, &t.CNCIIP)
	if err != nil {
		glog.Warning("unable to retrieve tenant from tenants")
		if err == sql.ErrNoRows {
			// not an error, it's just not there.
			err = nil
		}
		return nil, err
	}

	t.Resources, err = ds.getTenantResources(id)

	err = ds.getTenantNetwork(t)
	if err != nil {
		glog.V(2).Info(err)
	}

	t.instances, err = ds.getTenantInstances(t.Id)

	return t, err
}

func (ds *sqliteDB) getWorkloadNoCache(id string) (*workload, error) {
	datastore := ds.db

	query := `SELECT id,
			 description,
			 filename,
			 fw_type,
			 vm_type,
			 image_id,
			 image_name
		  FROM workload_template
		  WHERE id = ?`

	work := new(workload)

	var VMType string

	err := datastore.QueryRow(query, id).Scan(&work.Id, &work.Description, &work.filename, &work.FWType, &VMType, &work.ImageID, &work.ImageName)
	if err != nil {
		return nil, err
	}

	work.VMType = payloads.Hypervisor(VMType)

	work.Config, err = ds.getConfigNoCache(id)
	if err != nil {
		return nil, err
	}

	work.Defaults, err = ds.getWorkloadDefaults(id)
	if err != nil {
		return nil, err
	}

	return work, nil
}

func (ds *sqliteDB) getWorkloadsNoCache() ([]*workload, error) {
	var workloads []*workload

	datastore := ds.db

	query := `SELECT id,
			 description,
			 filename,
			 fw_type,
			 vm_type,
			 image_id,
			 image_name
		  FROM workload_template
		  WHERE internal = 0`

	rows, err := datastore.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		wl := new(workload)

		var VMType string

		err = rows.Scan(&wl.Id, &wl.Description, &wl.filename, &wl.FWType, &VMType, &wl.ImageID, &wl.ImageName)
		if err != nil {
			return nil, err
		}

		wl.Config, err = ds.getConfigNoCache(wl.Id)
		if err != nil {
			return nil, err
		}

		wl.Defaults, err = ds.getWorkloadDefaults(wl.Id)
		if err != nil {
			return nil, err
		}

		wl.VMType = payloads.Hypervisor(VMType)

		workloads = append(workloads, wl)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return workloads, nil
}

func (ds *sqliteDB) updateTenant(t *tenant) (err error) {
	db := ds.getTableDB("tenants")

	cmd := fmt.Sprintf("UPDATE tenants SET cnci_id = '%s', cnci_mac = '%s', cnci_ip = '%s' WHERE id = '%s'", t.CNCIID, t.CNCIMAC, t.CNCIIP, t.Id)

	ds.dbLock.Lock()
	err = ds.exec(db, cmd)
	ds.dbLock.Unlock()

	return
}

func (ds *sqliteDB) getTenantsNoCache() ([]*tenant, error) {
	var tenants []*tenant

	datastore := ds.getTableDB("tenants")

	query := `SELECT	tenants.id,
				tenants.name,
				tenants.cnci_id,
				tenants.cnci_mac,
				tenants.cnci_ip
		  FROM tenants `

	rows, err := datastore.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id sql.NullString
		var name sql.NullString
		var cnciID sql.NullString
		var cnciMAC sql.NullString
		var cnciIP sql.NullString

		t := new(tenant)
		err = rows.Scan(&id, &name, &cnciID, &cnciMAC, &cnciIP)
		if err != nil {
			return nil, err
		}

		if id.Valid {
			t.Id = id.String
		}
		if name.Valid {
			t.Name = name.String
		}
		if cnciID.Valid {
			t.CNCIID = cnciID.String
		}
		if cnciMAC.Valid {
			t.CNCIMAC = cnciMAC.String
		}
		if cnciIP.Valid {
			t.CNCIIP = cnciIP.String
		}

		t.Resources, err = ds.getTenantResources(t.Id)
		if err != nil {
			return nil, err
		}

		err = ds.getTenantNetwork(t)
		if err != nil {
			return nil, err
		}

		t.instances, err = ds.getTenantInstances(t.Id)
		if err != nil {
			return nil, err
		}

		tenants = append(tenants, t)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tenants, nil
}

func (ds *sqliteDB) claimTenantIP(tenantID string, subnetInt int, rest int) (err error) {
	datastore := ds.getTableDB("tenant_network")
	ds.dbLock.Lock()
	tx, err := datastore.Begin()
	if err != nil {
		ds.dbLock.Unlock()
		return
	}

	cmd := `INSERT INTO tenant_network VALUES('%s', %d, %d);`
	str := fmt.Sprintf(cmd, tenantID, subnetInt, rest)
	_, err = tx.Exec(str)
	if err != nil {
		glog.Warning(cmd, err)
		tx.Rollback()
		ds.dbLock.Unlock()
		return
	}

	tx.Commit()
	ds.dbLock.Unlock()

	return
}

func (ds *sqliteDB) releaseTenantIP(tenantID string, subnetInt int, rest int) (err error) {
	datastore := ds.getTableDB("tenant_network")

	cmd := fmt.Sprintf("DELETE FROM tenant_network WHERE tenant_id = '%s' AND subnet = %d AND rest = %d", tenantID, subnetInt, rest)

	ds.dbLock.Lock()
	err = ds.exec(datastore, cmd)
	ds.dbLock.Unlock()

	return
}

func (ds *sqliteDB) getTenantNetwork(tenant *tenant) (err error) {
	tenant.network = make(map[int]map[int]bool)

	// serialize
	ds.dbLock.Lock()
	datastore := ds.getTableDB("tenant_network")
	tx, err := datastore.Begin()
	if err != nil {
		ds.dbLock.Unlock()
		return
	}

	// get all subnet,rest values for this tenant
	query := `SELECT subnet, rest
		  FROM tenant_network
		  WHERE tenant_id = ?`
	rows, err := tx.Query(query, tenant.Id)
	if err != nil {
		glog.Warning(err)
		tx.Rollback()
		ds.dbLock.Unlock()
		return
	}
	defer rows.Close()

	for rows.Next() {
		var subnetInt uint16
		var rest uint8

		err = rows.Scan(&subnetInt, &rest)
		if err != nil {
			glog.Warning(err)
			tx.Rollback()
			ds.dbLock.Unlock()
			return
		}
		sub, ok := tenant.network[int(subnetInt)]
		if !ok {
			sub = make(map[int]bool)
			tenant.network[int(subnetInt)] = sub
		}
		/* Only add to the subnet list for the first host */
		if len(tenant.network[int(subnetInt)]) == 0 {
			tenant.subnets = append(tenant.subnets, int(subnetInt))
		}
		tenant.network[int(subnetInt)][int(rest)] = true

	}
	tx.Commit()
	ds.dbLock.Unlock()
	return
}

func (ds *sqliteDB) getInstances() (instances []*types.Instance, err error) {
	datastore := ds.getTableDB("instances")

	ds.tdbLock.RLock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}

	query := `
	WITH latest AS
	(
		SELECT 	max(tdb.instance_statistics.timestamp),
			tdb.instance_statistics.instance_id,
			tdb.instance_statistics.state,
			tdb.instance_statistics.ssh_ip,
			tdb.instance_statistics.ssh_port,
			tdb.instance_statistics.node_id
		FROM tdb.instance_statistics
		GROUP BY tdb.instance_statistics.instance_id
	)
	SELECT	instances.id,
		instances.tenant_id,
		IFNULL(latest.state, "pending") AS state,
		workload_id,
		IFNULL(latest.ssh_ip, "Not Assigned") as ssh_ip,
		latest.ssh_port as ssh_port,
		IFNULL(latest.node_id, "Not Assigned") as node_id,
		mac_address,
		ip
	FROM instances
	LEFT JOIN latest
	ON instances.id = latest.instance_id
	`

	rows, err := tx.Query(query)
	if err != nil {
		return nil, err
		ds.tdbLock.RUnlock()
		tx.Rollback()
	}
	defer rows.Close()

	for rows.Next() {
		var i types.Instance

		var sshPort sql.NullInt64

		err = rows.Scan(&i.Id, &i.TenantId, &i.State, &i.WorkloadId, &i.SSHIP, &sshPort, &i.NodeId, &i.MACAddress, &i.IPAddress)
		if err != nil {
			tx.Rollback()
			ds.tdbLock.RUnlock()
			return nil, err
		}

		if sshPort.Valid {
			i.SSHPort = int(sshPort.Int64)
		}

		instances = append(instances, &i)
	}
	if err = rows.Err(); err != nil {
		tx.Rollback()
		ds.tdbLock.RUnlock()
		return nil, err
	}

	tx.Commit()

	ds.tdbLock.RUnlock()

	return instances, nil
}

func (ds *sqliteDB) getTenantInstances(tenantID string) (instances map[string]*types.Instance, err error) {
	datastore := ds.getTableDB("instances")

	ds.tdbLock.RLock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}

	query := `
	WITH latest AS
	(
		SELECT 	max(tdb.instance_statistics.timestamp),
			tdb.instance_statistics.instance_id,
			tdb.instance_statistics.state,
			tdb.instance_statistics.ssh_ip,
			tdb.instance_statistics.ssh_port,
			tdb.instance_statistics.node_id
		FROM tdb.instance_statistics
		GROUP BY tdb.instance_statistics.instance_id
	)
	SELECT	instances.id,
		instances.tenant_id,
		IFNULL(latest.state, "pending") AS state,
		IFNULL(latest.ssh_ip, "Not Assigned") AS ssh_ip,
		latest.ssh_port AS ssh_port,
		workload_id,
		latest.node_id,
		mac_address,
		ip
	FROM instances
	LEFT JOIN latest
	ON instances.id = latest.instance_id
	WHERE instances.tenant_id = ?
	`

	rows, err := tx.Query(query, tenantID)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	instances = make(map[string]*types.Instance)
	for rows.Next() {
		var id sql.NullString
		var tenantID sql.NullString
		var state sql.NullString
		var workloadID sql.NullString
		var nodeID sql.NullString
		var macAddress sql.NullString
		var ipAddress sql.NullString
		var sshIP sql.NullString
		var sshPort sql.NullInt64

		i := new(types.Instance)
		err = rows.Scan(&id, &tenantID, &state, &sshIP, &sshPort, &workloadID, &nodeID, &macAddress, &ipAddress)
		if err != nil {
			tx.Rollback()
			ds.tdbLock.RUnlock()
			return nil, err
		}

		if id.Valid {
			i.Id = id.String
		}
		if tenantID.Valid {
			i.TenantId = tenantID.String
		}
		if state.Valid {
			i.State = state.String
		}
		if workloadID.Valid {
			i.WorkloadId = workloadID.String
		}
		if macAddress.Valid {
			i.MACAddress = macAddress.String
		}
		if ipAddress.Valid {
			i.IPAddress = ipAddress.String
		}
		if nodeID.Valid {
			i.NodeId = nodeID.String
		}
		if sshIP.Valid {
			i.SSHIP = sshIP.String
		}
		if sshPort.Valid {
			i.SSHPort = int(sshPort.Int64)
		}
		instances[i.Id] = i
	}
	if err = rows.Err(); err != nil {
		tx.Rollback()
		ds.tdbLock.RUnlock()
		return nil, err
	}
	tx.Commit()

	ds.tdbLock.RUnlock()

	return instances, nil
}

func (ds *sqliteDB) addInstance(instance *types.Instance) (err error) {
	ds.dbLock.Lock()

	err = ds.create("instances", instance.Id, instance.TenantId, instance.WorkloadId, instance.MACAddress, instance.IPAddress)

	ds.dbLock.Unlock()

	return
}

func (ds *sqliteDB) removeInstance(instanceID string) (err error) {
	cmd := `DELETE FROM instances WHERE id = '%s';`
	str := fmt.Sprintf(cmd, instanceID)

	ds.dbLock.Lock()
	err = ds.exec(ds.getTableDB("instances"), str)
	ds.dbLock.Unlock()

	return
}

func (ds *sqliteDB) addUsage(instanceID string, usage map[string]int) {
	cmd := `INSERT INTO usage (instance_id, resource_id, value)
		SELECT '%s', resources.id, %d FROM resources
		WHERE name = '%s';`

	for key, val := range usage {
		str := fmt.Sprintf(cmd, instanceID, val, key)
		ds.dbLock.Lock()
		err := ds.exec(ds.getTableDB("usage"), str)
		ds.dbLock.Unlock()
		if err != nil {
			glog.V(2).Info(err)
			// but keep going
		}
	}
}

func (ds *sqliteDB) deleteUsageNoCache(instanceID string) (err error) {
	cmd := fmt.Sprintf("DELETE FROM usage WHERE instance_id = '%s';", instanceID)

	ds.dbLock.Lock()
	err = ds.exec(ds.getTableDB("usage"), cmd)
	ds.dbLock.Unlock()

	return
}

func (ds *sqliteDB) addNodeStatDB(stat payloads.Stat) (err error) {
	cmd := `INSERT INTO node_statistics (node_id, mem_total_mb, mem_available_mb, disk_total_mb, disk_available_mb, load, cpus_online)
		VALUES('%s', %d, %d, %d, %d, %d, %d);`

	str := fmt.Sprintf(cmd, stat.NodeUUID, stat.MemTotalMB, stat.MemAvailableMB, stat.DiskTotalMB, stat.DiskAvailableMB, stat.Load, stat.CpusOnline)

	ds.tdbLock.Lock()

	err = ds.exec(ds.getTableDB("node_statistics"), str)

	ds.tdbLock.Unlock()

	return
}

func (ds *sqliteDB) addInstanceStatsDB(stats []payloads.InstanceStat, nodeID string) (err error) {
	datastore := ds.getTableDB("instance_statistics")

	ds.tdbLock.Lock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.Unlock()
		return
	}

	cmd := `INSERT INTO instance_statistics (instance_id, memory_usage_mb, disk_usage_mb, cpu_usage, state, node_id, ssh_ip, ssh_port)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(cmd)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.Unlock()
		return
	}
	defer stmt.Close()

	for index := range stats {
		stat := stats[index]

		_, err = stmt.Exec(stat.InstanceUUID, stat.MemoryUsageMB, stat.DiskUsageMB, stat.CPUUsage, stat.State, nodeID, stat.SSHIP, stat.SSHPort)
		if err != nil {
			glog.Warning(err)
			// but keep going
		}

	}
	tx.Commit()

	ds.tdbLock.Unlock()

	return
}

func (ds *sqliteDB) addFrameStat(stat payloads.FrameTrace) (err error) {
	datastore := ds.getTableDB("frame_statistics")

	ds.tdbLock.Lock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.Unlock()
		return
	}

	cmd := `INSERT INTO frame_statistics (label, type, operand, start_timestamp, end_timestamp)
		VALUES('%s', '%s', '%s', '%s', '%s')`
	str := fmt.Sprintf(cmd, stat.Label, stat.Type, stat.Operand, stat.StartTimestamp, stat.EndTimestamp)
	_, err = tx.Exec(str)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.Unlock()
		return
	}

	var id int
	err = tx.QueryRow("SELECT last_insert_rowid();").Scan(&id)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.Unlock()
		return
	}

	for index := range stat.Nodes {
		t := stat.Nodes[index]
		cmd := `INSERT INTO trace_data (frame_id, ssntp_uuid, tx_timestamp, rx_timestamp)
			VALUES(%d, '%s', '%s', '%s');`
		str := fmt.Sprintf(cmd, id, t.SSNTPUUID, t.TxTimestamp, t.RxTimestamp)
		_, err = tx.Exec(str)
		if err != nil {
			tx.Rollback()
			ds.tdbLock.Unlock()
			return
		}
	}
	tx.Commit()
	ds.tdbLock.Unlock()
	return
}

// GetEventLog retrieves all the log entries stored in the datastore.
func (ds *sqliteDB) getEventLog() (logEntries []*types.LogEntry, err error) {
	datastore := ds.getTableDB("log")

	ds.tdbLock.RLock()

	rows, err := datastore.Query("SELECT timestamp, tenant_id, type, message FROM log")
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	logEntries = make([]*types.LogEntry, 0)
	for rows.Next() {
		var e types.LogEntry
		err = rows.Scan(&e.Timestamp, &e.TenantId, &e.EventType, &e.Message)
		if err != nil {
			ds.tdbLock.RUnlock()
			return nil, err
		}
		logEntries = append(logEntries, &e)
	}

	ds.tdbLock.RUnlock()

	return logEntries, err
}

// GetNodeSummary provides a summary the state and count of instances running per node.
func (ds *sqliteDB) getNodeSummary() (Summary []*types.NodeSummary, err error) {
	datastore := ds.getTableDB("instance_statistics")
	ds.tdbLock.RLock()
	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.RUnlock()
		return
	}

	query := `
WITH instances AS
(
	WITH latest AS
	(
		SELECT 	max(timestamp),
			instance_id,
			node_id,
			state
		FROM instance_statistics
		GROUP BY instance_id
	)
	SELECT db.instances.id AS instance_id,
	       IFNULL(latest.state, "pending") AS state,
	       IFNULL(latest.node_id, "Not Assigned") AS node_id
	FROM db.instances
	LEFT JOIN latest
	ON db.instances.id = latest.instance_id
),
total_instances AS
(
	SELECT 	IFNULL(instances.node_id, "Not Assigned to Node") AS node_id,
		count(instances.instance_id) AS total
	FROM instances
	GROUP BY node_id
),
total_running AS
(
	SELECT	instances.node_id AS node_id,
		count(instances.instance_id) AS total
	FROM instances
	WHERE state='running'
	GROUP BY node_id
),
total_pending AS
(
	SELECT	instances.node_id AS node_id,
		count(instances.instance_id) AS total
	FROM instances
	WHERE state='pending'
	GROUP BY node_id
),
total_exited AS
(
	SELECT	instances.node_id,
		count(instances.instance_id) AS total
	FROM instances
	WHERE state='exited'
	GROUP BY node_id
)
SELECT	total_instances.node_id,
	total_instances.total,
        IFNULL(total_running.total, 0),
	IFNULL(total_pending.total, 0),
	IFNULL(total_exited.total, 0)
FROM total_instances
LEFT JOIN total_running
ON total_instances.node_id = total_running.node_id
LEFT JOIN total_pending
ON total_instances.node_id = total_pending.node_id
LEFT JOIN total_exited
ON total_instances.node_id = total_exited.node_id
`

	rows, err := tx.Query(query)
	if err != nil {
		glog.V(2).Info("Failed to get Node Summary: ", err)
		tx.Rollback()
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	Summary = make([]*types.NodeSummary, 0)
	for rows.Next() {
		var n types.NodeSummary
		err = rows.Scan(&n.NodeId, &n.TotalInstances, &n.TotalRunningInstances, &n.TotalPendingInstances, &n.TotalPausedInstances)
		if err != nil {
			tx.Rollback()
			ds.tdbLock.RUnlock()
			return nil, err
		}
		Summary = append(Summary, &n)
	}
	tx.Commit()
	ds.tdbLock.RUnlock()

	return Summary, err
}

// GetFrameStatistics will return trace data by label id.
func (ds *sqliteDB) GetFrameStatistics(label string) (stats []types.FrameStat, err error) {
	ds.tdbLock.RLock()
	query := `WITH total AS
		 (
			SELECT	id,
				start_timestamp,
				end_timestamp,
				(julianday(end_timestamp) - julianday(start_timestamp)) * 24 * 60 * 60 AS total_elapsed
			 FROM frame_statistics
			 WHERE label = ?
		 ),
		 total_start AS
		 (
			SELECT	trace_data.frame_id,
				trace_data.ssntp_uuid,
				(julianday(trace_data.tx_timestamp) - julianday(total.start_timestamp)) * 24 * 60 * 60 AS total_elapsed
			FROM trace_data
			JOIN total
			WHERE rx_timestamp = '' and trace_data.frame_id = total.id
		),
		total_end AS
		(
			SELECT	trace_data.frame_id,
				trace_data.ssntp_uuid,
				(julianday(total.end_timestamp) - julianday(trace_data.rx_timestamp)) * 24 * 60 * 60 AS total_elapsed
			FROM trace_data
			JOIN total
			WHERE tx_timestamp = '' and trace_data.frame_id = total.id
		),
		total_per_node AS
		(
			SELECT	trace_data.frame_id,
				trace_data.ssntp_uuid,
				(julianday(trace_data.tx_timestamp) - julianday(trace_data.rx_timestamp)) * 24 * 60 * 60 AS total_elapsed
			FROM trace_data
			WHERE tx_timestamp != '' and rx_timestamp != ''
		)
		SELECT	total_end.ssntp_uuid,
			total.total_elapsed,
			total_start.total_elapsed,
			total_end.total_elapsed,
			total_per_node.total_elapsed
		FROM total
		LEFT JOIN total_start
		ON total.id = total_start.frame_id
		LEFT JOIN total_end
		ON total_start.frame_id = total_end.frame_id
		LEFT JOIN total_per_node
		ON total_start.frame_id = total_per_node.frame_id
		ORDER BY total.start_timestamp;`

	datastore := ds.getTableDB("frame_statistics")

	rows, err := datastore.Query(query, label)
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	stats = make([]types.FrameStat, 0)
	for rows.Next() {
		var stat types.FrameStat
		var uuid sql.NullString
		var controllerTime sql.NullFloat64
		var launcherTime sql.NullFloat64
		var schedulerTime sql.NullFloat64
		var totalTime sql.NullFloat64
		err = rows.Scan(&uuid, &totalTime, &controllerTime, &launcherTime, &schedulerTime)
		if err != nil {
			ds.tdbLock.RUnlock()
			return
		}
		if uuid.Valid {
			stat.ID = uuid.String
		}
		if controllerTime.Valid {
			stat.ControllerTime = controllerTime.Float64
		}
		if launcherTime.Valid {
			stat.LauncherTime = launcherTime.Float64
		}
		if schedulerTime.Valid {
			stat.SchedulerTime = schedulerTime.Float64
		}
		if totalTime.Valid {
			stat.TotalElapsedTime = totalTime.Float64
		}
		stats = append(stats, stat)
	}
	ds.tdbLock.RUnlock()

	return stats, err
}

// GetBatchFrameSummary will retieve the count of traces we have for a specific label
func (ds *sqliteDB) getBatchFrameSummary() (stats []types.BatchFrameSummary, err error) {
	datastore := ds.getTableDB("frame_statistics")

	ds.tdbLock.RLock()

	query := `SELECT label, count(id)
		  FROM frame_statistics
		  GROUP BY label;`

	rows, err := datastore.Query(query)
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	stats = make([]types.BatchFrameSummary, 0)
	for rows.Next() {
		var stat types.BatchFrameSummary
		err = rows.Scan(&stat.BatchID, &stat.NumInstances)
		if err != nil {
			ds.tdbLock.RUnlock()
			return
		}
		stats = append(stats, stat)
	}
	ds.tdbLock.RUnlock()

	return stats, err
}

// GetBatchFrameStatistics will show individual trace data per instance for a batch of trace data.
// The batch is identified by the label.
func (ds *sqliteDB) getBatchFrameStatistics(label string) (stats []types.BatchFrameStat, err error) {
	datastore := ds.getTableDB("frame_statistics")

	query := `WITH total AS
		 (
			SELECT	id,
				start_timestamp,
				end_timestamp,
				(julianday(end_timestamp) - julianday(start_timestamp)) * 24 * 60 * 60 AS total_elapsed
			FROM frame_statistics
			WHERE label = ?
		),
		total_start AS
		(
			SELECT	trace_data.frame_id,
				trace_data.ssntp_uuid,
				(julianday(trace_data.tx_timestamp) - julianday(total.start_timestamp)) * 24 * 60 * 60 AS total_elapsed
			FROM trace_data
			JOIN total
			WHERE rx_timestamp = '' and trace_data.frame_id = total.id
		),
		total_end AS
		(
			SELECT 	trace_data.frame_id,
				trace_data.ssntp_uuid,
				(julianday(total.end_timestamp) - julianday(trace_data.rx_timestamp)) * 24 * 60 * 60 AS total_elapsed
			FROM trace_data
			JOIN total
			WHERE tx_timestamp = '' and trace_data.frame_id = total.id
		),
		total_per_node AS
		(
			SELECT	trace_data.frame_id,
				trace_data.ssntp_uuid,
				(julianday(trace_data.tx_timestamp) - julianday(trace_data.rx_timestamp)) * 24 * 60 *60 AS total_elapsed
			FROM trace_data
			WHERE tx_timestamp != '' and rx_timestamp != ''
		),
		diffs AS
		(
			SELECT 	total.id AS id,
				total.total_elapsed AS total_elapsed,
				total_start.total_elapsed AS controller_elapsed,
				total_end.total_elapsed AS launcher_elapsed,
				total_per_node.total_elapsed AS scheduler_elapsed
			FROM total
			LEFT JOIN total_start
			ON total.id = total_start.frame_id
			LEFT JOIN total_end
			ON total_start.frame_id = total_end.frame_id
			LEFT JOIN total_per_node
			ON total_start.frame_id = total_per_node.frame_id
		),
		averages AS
		(
			SELECT	avg(diffs.total_elapsed) AS avg_total_elapsed,
				avg(diffs.controller_elapsed) AS avg_controller,
				avg(diffs.launcher_elapsed) AS avg_launcher,
				avg(diffs.scheduler_elapsed) AS avg_scheduler
			FROM diffs
		),
		variance AS
		(
			SELECT	avg((total_start.total_elapsed - averages.avg_controller) * (total_start.total_elapsed - averages.avg_controller)) AS controller,
				avg((total_end.total_elapsed - averages.avg_launcher) * (total_end.total_elapsed - averages.avg_launcher)) AS launcher,
				avg((total_per_node.total_elapsed - averages.avg_scheduler) * (total_per_node.total_elapsed - averages.avg_scheduler)) AS scheduler
			FROM total_start
			LEFT JOIN total_end
			ON total_start.frame_id = total_end.frame_id
			LEFT JOIN total_per_node
			ON total_start.frame_id = total_per_node.frame_id
			JOIN averages
		)
		SELECT	count(total.id) AS num_instances,
			(julianday(max(total.end_timestamp)) - julianday(min(total.start_timestamp))) * 24 * 60 * 60 AS total_elapsed,
			averages.avg_total_elapsed AS average_total_elapsed,
			averages.avg_controller AS average_controller_elapsed,
			averages.avg_launcher AS average_launcher_elapsed,
			averages.avg_scheduler AS average_scheduler_elapsed,
			variance.controller AS controller_variance,
			variance.launcher AS launcher_variance,
			variance.scheduler AS scheduler_variance
		FROM variance
		JOIN total
		JOIN averages;`
	ds.tdbLock.RLock()
	rows, err := datastore.Query(query, label)
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	stats = make([]types.BatchFrameStat, 0)
	for rows.Next() {
		var stat types.BatchFrameStat
		var numInstances sql.NullInt64
		var totalElapsed sql.NullFloat64
		var averageElapsed sql.NullFloat64
		var averageControllerElapsed sql.NullFloat64
		var averageLauncherElapsed sql.NullFloat64
		var averageSchedulerElapsed sql.NullFloat64
		var varianceController sql.NullFloat64
		var varianceLauncher sql.NullFloat64
		var varianceScheduler sql.NullFloat64

		err = rows.Scan(&numInstances, &totalElapsed, &averageElapsed, &averageControllerElapsed, &averageLauncherElapsed, &averageSchedulerElapsed, &varianceController, &varianceLauncher, &varianceScheduler)
		if err != nil {
			ds.tdbLock.RUnlock()
			return
		}
		if numInstances.Valid {
			stat.NumInstances = int(numInstances.Int64)
		}
		if totalElapsed.Valid {
			stat.TotalElapsed = totalElapsed.Float64
		}
		if averageElapsed.Valid {
			stat.AverageElapsed = averageElapsed.Float64
		}
		if averageControllerElapsed.Valid {
			stat.AverageControllerElapsed = averageControllerElapsed.Float64
		}
		if averageLauncherElapsed.Valid {
			stat.AverageLauncherElapsed = averageLauncherElapsed.Float64
		}
		if averageSchedulerElapsed.Valid {
			stat.AverageSchedulerElapsed = averageSchedulerElapsed.Float64
		}
		if varianceController.Valid {
			stat.VarianceController = varianceController.Float64
		}
		if varianceLauncher.Valid {
			stat.VarianceLauncher = varianceLauncher.Float64
		}
		if varianceScheduler.Valid {
			stat.VarianceScheduler = varianceScheduler.Float64
		}
		stats = append(stats, stat)
	}

	ds.tdbLock.RUnlock()

	return stats, err
}
