/*
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
*/

// Package datastore retrieves stores data for the ciao controller.
// This package caches most data in memory, and uses a sql
// database as persistent storage.
package datastore

import (
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/golang/glog"
	sqlite3 "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type userEventType string

const (
	userInfo  userEventType = "info"
	userWarn  userEventType = "warn"
	userError userEventType = "error"
)

type workload struct {
	types.Workload
	filename string
}

type tenant struct {
	types.Tenant
	network   map[int]map[int]bool
	subnets   []int
	instances map[string]*types.Instance
}

type node struct {
	types.Node
	instances map[string]*types.Instance
}

// Datastore provides context for the datastore package.
type Datastore struct {
	db            *sql.DB
	tdb           *sql.DB
	dbName        string
	tdbName       string
	tables        []persistentData
	tableInitPath string
	workloadsPath string
	dbLock        *sync.Mutex
	tdbLock       *sync.RWMutex

	cnciAddedChans map[string]chan bool
	cnciAddedLock  *sync.Mutex

	nodeLastStat     map[string]payloads.CiaoComputeNode
	nodeLastStatLock *sync.RWMutex

	instanceLastStat     map[string]payloads.CiaoServerStats
	instanceLastStatLock *sync.RWMutex

	tenants     map[string]*tenant
	tenantsLock *sync.RWMutex
	allSubnets  map[int]bool

	workloads     map[string]*workload
	workloadsLock *sync.RWMutex

	nodes     map[string]*node
	nodesLock *sync.RWMutex

	instances     map[string]*types.Instance
	instancesLock *sync.RWMutex

	tenantUsage     map[string][]payloads.CiaoUsage
	tenantUsageLock *sync.RWMutex
}

type persistentData interface {
	Init() error
	Populate() error
	Create(...string) error
	Name() string
	DB() *sql.DB
}

type namedData struct {
	ds   *Datastore
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

func (ds *Datastore) exec(db *sql.DB, cmd string) (err error) {
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

func (ds *Datastore) create(tableName string, record ...interface{}) (err error) {
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

func (ds *Datastore) getTableDB(name string) *sql.DB {
	for _, table := range ds.tables {
		n := table.Name()
		if n == name {
			return table.DB()
		}
	}
	return nil
}

// Init initializes the private data for the Datastore object.
// The sql tables are populated with initial data from csv
// files if this is the first time the database has been
// created.  The datastore caches are also filled.
func (ds *Datastore) Init(tableInitPath string, workloadsPath string) (err error) {
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

	ds.tableInitPath = tableInitPath
	ds.workloadsPath = workloadsPath

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

	ds.cnciAddedChans = make(map[string]chan bool)
	ds.cnciAddedLock = &sync.Mutex{}

	ds.nodeLastStat = make(map[string]payloads.CiaoComputeNode)
	ds.nodeLastStatLock = &sync.RWMutex{}

	ds.instanceLastStat = make(map[string]payloads.CiaoServerStats)
	ds.instanceLastStatLock = &sync.RWMutex{}

	// warning, do not use the tenant cache to get
	// networking information right now.  that is not
	// updated, just the resources
	ds.tenants = make(map[string]*tenant)
	ds.allSubnets = make(map[int]bool)
	ds.tenantsLock = &sync.RWMutex{}

	// cache all our instances prior to getting tenants
	ds.instancesLock = &sync.RWMutex{}
	ds.instances = make(map[string]*types.Instance)

	instances, err := ds.GetAllInstances()
	if err != nil {
		glog.Warning(err)
	} else {
		for i := range instances {
			ds.instances[instances[i].Id] = instances[i]
		}
	}

	// cache our current tenants into a map that we can
	// quickly index
	tenants, err := ds.getTenants()
	if err == nil {
		for i := range tenants {
			ds.tenants[tenants[i].Id] = tenants[i]
		}
	}

	// cache the workloads into a map so that we can
	// quickly index
	ds.workloadsLock = &sync.RWMutex{}
	ds.workloads = make(map[string]*workload)
	workloads, err := ds.getWorkloads()
	if err != nil {
		glog.Warning(err)
	} else {
		for i := range workloads {
			ds.workloads[workloads[i].Id] = workloads[i]
		}
	}

	ds.nodesLock = &sync.RWMutex{}
	ds.nodes = make(map[string]*node)

	for key, i := range ds.instances {
		n, ok := ds.nodes[i.NodeId]
		if !ok {
			newNode := types.Node{
				ID: i.NodeId,
			}
			n = &node{
				Node:      newNode,
				instances: make(map[string]*types.Instance),
			}
			ds.nodes[i.NodeId] = n
		}
		ds.nodes[i.NodeId].instances[key] = i
	}

	ds.tenantUsage = make(map[string][]payloads.CiaoUsage)
	ds.tenantUsageLock = &sync.RWMutex{}

	return
}

// Connect creates two sqlite3 databases.  One database is for
// persistent state that needs to be restored on restart, the
// other is for transient data that does not need to be restored
// on restart.
func (ds *Datastore) Connect(persistentURI string, transientURI string) (err error) {
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
func (ds *Datastore) Disconnect() {
	ds.db.Close()
}

func (ds *Datastore) logEvent(tenantID string, eventType string, message string) error {
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
func (ds *Datastore) ClearLog() error {
	db := ds.getTableDB("log")

	ds.tdbLock.Lock()

	err := ds.exec(db, "DELETE FROM log")
	if err != nil {
		glog.V(2).Info("could not clear log: ", err)
	}

	ds.tdbLock.Unlock()

	return err
}

// AddTenantChan allows a caller to pass in a channel for CNCI Launch status.
// When a CNCI has been added to the datastore and a channel exists,
// success will be indicated on the channel.  If a CNCI failure occurred
// and a channel exists, failure will be indicated on the channel.
func (ds *Datastore) AddTenantChan(c chan bool, tenantID string) {
	ds.cnciAddedLock.Lock()
	ds.cnciAddedChans[tenantID] = c
	ds.cnciAddedLock.Unlock()
}

// GetCNCIWorkloadID returns the UUID of the workload template
// for the CNCI workload
func (ds *Datastore) GetCNCIWorkloadID() (id string, err error) {
	db := ds.getTableDB("workload_template")

	err = db.QueryRow("SELECT id FROM workload_template WHERE description = 'CNCI'").Scan(&id)
	if err != nil {
		return
	}
	return
}

func (ds *Datastore) getConfig(id string) (config string, err error) {
	// check our cache first
	ds.workloadsLock.RLock()
	wl := ds.workloads[id]
	ds.workloadsLock.RUnlock()

	var configFile string

	if wl == nil {
		db := ds.getTableDB("workload_template")

		err = db.QueryRow("SELECT filename FROM workload_template where id = ?", id).Scan(&configFile)

		if err != nil {
			return config, err
		}
	} else {
		return wl.Config, nil
	}

	path := fmt.Sprintf("%s/%s", ds.workloadsPath, configFile)
	bytes, err := ioutil.ReadFile(path)
	config = string(bytes)
	return config, err
}

func (ds *Datastore) getImageInfo(workloadID string) (imageID string, fwType string, err error) {
	// check the cache first
	ds.workloadsLock.RLock()
	wl := ds.workloads[workloadID]
	ds.workloadsLock.RUnlock()

	if wl != nil {
		return wl.ImageID, wl.FWType, nil
	}

	db := ds.getTableDB("workload_template")

	err = db.QueryRow("SELECT image_id, fw_type FROM workload_template where id = ?", workloadID).Scan(&imageID, &fwType)

	if err != nil {
		return
	}

	return
}

func (ds *Datastore) getWorkloadDefaults(id string) (defaults []payloads.RequestedResource, err error) {
	// check the cache first
	ds.workloadsLock.RLock()
	wl := ds.workloads[id]
	ds.workloadsLock.RUnlock()

	if wl != nil {
		return wl.Defaults, nil
	}

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

// AddLimit allows the caller to store a limt for a specific resource for a tenant.
func (ds *Datastore) AddLimit(tenantID string, resourceID int, limit int) (err error) {
	ds.dbLock.Lock()
	err = ds.create("limits", resourceID, tenantID, limit)
	ds.dbLock.Unlock()
	if err != nil {
		return
	}

	// update cache
	ds.tenantsLock.Lock()
	tenant := ds.tenants[tenantID]
	if tenant != nil {
		resources := tenant.Resources
		for i := range resources {
			if resources[i].Rtype == resourceID {
				resources[i].Limit = limit
				break
			}
		}
	}
	ds.tenantsLock.Unlock()
	return
}

func (ds *Datastore) getTenantResources(id string) (resources []*types.Resource, err error) {
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

func newHardwareAddr() (hw net.HardwareAddr, err error) {
	buf := make([]byte, 6)
	_, err = rand.Read(buf)
	if err != nil {
		return
	}

	// vnic creation seems to require not just the
	// bit 1 to be set, but the entire byte to be
	// set to 2.  Also, ensure that we get no
	// overlap with tenant mac addresses by not allowing
	// byte 1 to ever be zero.
	buf[0] = 2
	if buf[1] == 0 {
		buf[1] = 3
	}
	hw = net.HardwareAddr(buf)
	return
}

// AddTenant stores information about a tenant into the datastore.
// it creates a MAC address for the tenant network and makes sure
// that this new tenant is cached.
func (ds *Datastore) AddTenant(id string) (tenant *types.Tenant, err error) {
	hw, err := newHardwareAddr()
	if err != nil {
		glog.V(2).Info("error creating mac address", err)
		return
	}
	ds.dbLock.Lock()
	err = ds.create("tenants", id, "", "", hw.String(), "")
	ds.dbLock.Unlock()

	t, err := ds.getTenant(id)
	if err != nil || t == nil {
		glog.V(2).Info(err, " unable to get tenant: ", id)
	}

	ds.tenantsLock.Lock()
	ds.tenants[id] = t
	ds.tenantsLock.Unlock()

	return &t.Tenant, err
}

func (ds *Datastore) getTenantNoCache(id string) (t *tenant, err error) {
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

func (ds *Datastore) getTenant(id string) (t *tenant, err error) {
	// check cache first
	ds.tenantsLock.RLock()
	t = ds.tenants[id]
	ds.tenantsLock.RUnlock()

	if t != nil {
		return
	}

	return ds.getTenantNoCache(id)
}

// GetTenant returns details about a tenant referenced by the uuid
func (ds *Datastore) GetTenant(id string) (tenant *types.Tenant, err error) {
	// check cache first
	ds.tenantsLock.RLock()
	t := ds.tenants[id]
	ds.tenantsLock.RUnlock()

	if t != nil {
		return &t.Tenant, nil
	}

	query := `SELECT	tenants.id,
				tenants.name,
				tenants.cnci_id,
				tenants.cnci_mac,
				tenants.cnci_ip
		  FROM tenants
		  WHERE tenants.id = ?`
	datastore := ds.db

	row := datastore.QueryRow(query, id)

	tenant = new(types.Tenant)

	err = row.Scan(&tenant.Id, &tenant.Name, &tenant.CNCIID, &tenant.CNCIMAC, &tenant.CNCIIP)
	if err != nil {
		glog.Warning("unable to retrieve tenant from tenants")
		if err == sql.ErrNoRows {
			// not an error, it's just not there.
			err = nil
		}
		return nil, err
	}

	tenant.Resources, err = ds.getTenantResources(id)

	return tenant, err
}

func (ds *Datastore) getWorkload(id string) (*workload, error) {
	// check the cache first
	ds.workloadsLock.RLock()
	wl := ds.workloads[id]
	ds.workloadsLock.RUnlock()

	if wl != nil {
		return wl, nil
	}

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

	work.Config, err = ds.getConfig(id)
	if err != nil {
		return nil, err
	}

	work.Defaults, err = ds.getWorkloadDefaults(id)
	if err != nil {
		return nil, err
	}

	return work, nil
}

// GetWorkload returns details about a specific workload referenced by id
func (ds *Datastore) GetWorkload(id string) (*types.Workload, error) {
	// check the cache first
	ds.workloadsLock.RLock()
	wl := ds.workloads[id]
	ds.workloadsLock.RUnlock()

	if wl != nil {
		return &wl.Workload, nil
	}

	wl, err := ds.getWorkload(id)
	if err != nil {
		return nil, err
	}

	return &wl.Workload, nil
}

func (ds *Datastore) getWorkloads() ([]*workload, error) {
	var workloads []*workload

	// check the cache first
	ds.workloadsLock.RLock()
	if len(ds.workloads) > 0 {
		for _, wl := range ds.workloads {
			workloads = append(workloads, wl)
		}
		ds.workloadsLock.RUnlock()
		return workloads, nil
	}
	ds.workloadsLock.RUnlock()

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

		wl.Config, err = ds.getConfig(wl.Id)
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

// GetWorkloads returns all known tenant workloads
func (ds *Datastore) GetWorkloads() ([]*types.Workload, error) {
	var workloads []*types.Workload

	// check the cache first
	ds.workloadsLock.RLock()
	if len(ds.workloads) > 0 {
		for _, wl := range ds.workloads {
			workloads = append(workloads, &wl.Workload)
		}
		ds.workloadsLock.RUnlock()
		return workloads, nil
	}
	ds.workloadsLock.RUnlock()

	wls, err := ds.getWorkloads()
	if err != nil {
		return workloads, err
	}

	if len(wls) > 0 {
		for _, wl := range wls {
			workloads = append(workloads, &wl.Workload)
		}
	}

	return workloads, nil
}

func (ds *Datastore) getTenantCNCI(tenantID string) (cnciID string, cnciIP string, cnciMAC string, err error) {
	// check the cache
	ds.tenantsLock.RLock()
	t := ds.tenants[tenantID]
	ds.tenantsLock.RUnlock()

	if t != nil {
		return t.CNCIID, t.CNCIIP, t.CNCIMAC, nil
	}

	datastore := ds.db

	err = datastore.QueryRow("SELECT cnci_id, cnci_ip, cnci_mac FROM tenants WHERE tenants.id = ?", tenantID).Scan(&cnciID, &cnciIP, &cnciMAC)
	return
}

func (ds *Datastore) getTenantByCNCIMAC(cnciMAC string) (tenantID string, err error) {
	db := ds.getTableDB("tenants")

	err = db.QueryRow("SELECT id FROM tenants WHERE cnci_mac = ?", cnciMAC).Scan(&tenantID)
	return
}

// AddCNCIIP will associate a new IP address with an existing CNCI
// via the mac address
func (ds *Datastore) AddCNCIIP(cnciMAC string, ip string) (err error) {
	// update tenants cache
	ds.tenantsLock.Lock()
	tenantID, err := ds.getTenantByCNCIMAC(cnciMAC)
	if err != nil {
		ds.tenantsLock.Unlock()
		return
	}

	if ds.tenants[tenantID] != nil {
		ds.tenants[tenantID].CNCIIP = ip
	}
	ds.tenantsLock.Unlock()

	db := ds.getTableDB("tenants")
	cmd := fmt.Sprintf("UPDATE tenants SET cnci_ip = '%s' WHERE cnci_mac = '%s'", ip, cnciMAC)
	ds.dbLock.Lock()
	err = ds.exec(db, cmd)
	ds.dbLock.Unlock()
	if err != nil {
		glog.Warning("Failed to update CNCI IP")
	}

	ds.cnciAddedLock.Lock()

	c, ok := ds.cnciAddedChans[tenantID]
	if ok {
		delete(ds.cnciAddedChans, tenantID)
	}

	ds.cnciAddedLock.Unlock()

	if c != nil {
		c <- true
	}

	return
}

// AddTenantCNCI will associate a new CNCI instance with a specific tenant.
// The instanceID of the new CNCI instance and the MAC address of the new instance
// are stored in the sql database and updated in the cache.
func (ds *Datastore) AddTenantCNCI(tenantID string, instanceID string, mac string) (err error) {
	// update tenants cache
	ds.tenantsLock.Lock()
	if ds.tenants[tenantID] != nil {
		ds.tenants[tenantID].CNCIID = instanceID
		ds.tenants[tenantID].CNCIMAC = mac
	}
	ds.tenantsLock.Unlock()

	db := ds.getTableDB("tenants")
	cmd := fmt.Sprintf("UPDATE tenants SET cnci_id = '%s', cnci_mac = '%s' WHERE id = '%s'", instanceID, mac, tenantID)
	ds.dbLock.Lock()
	err = ds.exec(db, cmd)
	ds.dbLock.Unlock()

	return
}

func (ds *Datastore) removeTenantCNCI(tenantID string, cnciID string) (err error) {
	// update tenants cache
	ds.tenantsLock.Lock()
	if ds.tenants[tenantID] != nil {
		ds.tenants[tenantID].CNCIID = ""
		ds.tenants[tenantID].CNCIIP = ""
	}
	ds.tenantsLock.Unlock()

	db := ds.getTableDB("tenants")
	cmd := fmt.Sprintf("UPDATE tenants SET cnci_id = '', cnci_ip = '' WHERE cnci_id = '%s'", cnciID)
	ds.dbLock.Lock()
	err = ds.exec(db, cmd)
	ds.dbLock.Unlock()

	return
}

func (ds *Datastore) getCNCITenant(cnciID string) (tenantID string, err error) {
	db := ds.getTableDB("tenants")

	err = db.QueryRow("SELECT id FROM tenants WHERE cnci_id = ?", cnciID).Scan(&tenantID)
	return
}

func (ds *Datastore) isInstanceCNCI(instanceID string) (b bool, err error) {
	datastore := ds.getTableDB("tenants")

	var c int
	err = datastore.QueryRow("SELECT count(cnci_id) FROM tenants WHERE cnci_id = ?", instanceID).Scan(&c)
	b = (c > 0)
	return
}

func (ds *Datastore) getTenants() ([]*tenant, error) {
	var tenants []*tenant

	// check the cache first
	ds.tenantsLock.RLock()
	if len(ds.tenants) > 0 {
		for _, value := range ds.tenants {
			tenants = append(tenants, value)
		}
		ds.tenantsLock.RUnlock()
		return tenants, nil
	}
	ds.tenantsLock.RUnlock()

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

// GetAllTenants returns all the tenants from the datastore.
func (ds *Datastore) GetAllTenants() ([]*types.Tenant, error) {
	var tenants []*types.Tenant

	// check the cache first
	ds.tenantsLock.RLock()
	if len(ds.tenants) > 0 {
		for _, value := range ds.tenants {
			tenants = append(tenants, &value.Tenant)
		}
		ds.tenantsLock.RUnlock()
		return tenants, nil
	}
	ds.tenantsLock.RUnlock()

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

		t := new(types.Tenant)
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

		tenants = append(tenants, t)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tenants, nil
}

// ReleaseTenantIP will return an IP address previously allocated to the pool.
// Once a tenant IP address is released, it can be reassigned to another
// instance.
func (ds *Datastore) ReleaseTenantIP(tenantID string, ip string) (err error) {
	datastore := ds.getTableDB("tenant_network")
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return errors.New("Invalid IPv4 Address")
	}

	ipBytes := ipAddr.To4()
	if ipBytes == nil {
		return errors.New("Unable to convert ip to bytes")
	}
	subnetInt := binary.BigEndian.Uint16(ipBytes[1:3])

	// clear from cache
	ds.tenantsLock.Lock()
	if ds.tenants[tenantID] != nil {
		ds.tenants[tenantID].network[int(subnetInt)][int(ipBytes[3])] = false
	}
	ds.tenantsLock.Unlock()

	cmd := fmt.Sprintf("DELETE FROM tenant_network WHERE tenant_id = '%s' AND subnet = %d AND rest = %d", tenantID, subnetInt, int(ipBytes[3]))
	ds.dbLock.Lock()
	err = ds.exec(datastore, cmd)
	ds.dbLock.Unlock()

	return
}

func (ds *Datastore) getTenantNetwork(tenant *tenant) (err error) {
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

// AllocateTenantIP will find a free IP address within a tenant network.
// For now we make each tenant have unique subnets even though it
// isn't actually needed because of a docker issue.
func (ds *Datastore) AllocateTenantIP(tenantID string) (ip net.IP, err error) {
	var subnetInt uint16
	subnetInt = 0

	ds.tenantsLock.Lock()
	network := ds.tenants[tenantID].network
	subnets := ds.tenants[tenantID].subnets

	// find any subnet assigned to this tenant with available addresses
	sort.Ints(subnets)
	for _, k := range subnets {
		if len(network[k]) < 253 {
			subnetInt = uint16(k)
		}
	}

	var subnetBytes = []byte{16, 0}
	if subnetInt == 0 {
		i := binary.BigEndian.Uint16(subnetBytes)

		for {
			// sub, ok := network[int(i)]
			// for now, prevent overlapping subnets
			// due to bug in docker.
			ok := ds.allSubnets[int(i)]
			if !ok {
				sub := make(map[int]bool)
				network[int(i)] = sub

				// claim so no one else can use it
				ds.allSubnets[int(i)] = true

				break
			}
			if subnetBytes[1] == 255 {
				if subnetBytes[0] == 31 {
					// out of possible subnets
					glog.Warning("Out of Subnets")
					ds.tenantsLock.Unlock()
					return nil, errors.New("Out of subnets")
				}
				subnetBytes[0]++
				subnetBytes[1] = 0
			} else {
				subnetBytes[1]++
			}
			i = binary.BigEndian.Uint16(subnetBytes)
		}
		subnetInt = i
		ds.tenants[tenantID].subnets = append(subnets, int(subnetInt))
	} else {
		binary.BigEndian.PutUint16(subnetBytes, subnetInt)
	}

	hosts := network[int(subnetInt)]

	rest := 2
	for {
		if hosts[rest] == false {
			hosts[rest] = true
			break
		}

		if rest == 255 {
			// this should never happen
			glog.Warning("ran out of host numbers")
			ds.tenantsLock.Unlock()
			return nil, errors.New("rand out of host numbers")
		}
		rest++
	}

	// at this point we have a subnet and host number, we should
	// claim this in the datastore
	go func(subnetInt uint16, rest int) {
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
	}(subnetInt, rest)

	ds.tenantsLock.Unlock()

	// convert to IP type.
	next := net.IPv4(172, subnetBytes[0], subnetBytes[1], byte(rest))
	return next, err
}

// GetAllInstances retrieves all instances out of the datastore.
func (ds *Datastore) GetAllInstances() (instances []*types.Instance, err error) {
	// always get from cache
	ds.instancesLock.RLock()
	if len(ds.instances) > 0 {
		for _, val := range ds.instances {
			instances = append(instances, val)
		}
		ds.instancesLock.RUnlock()
		return
	}
	ds.instancesLock.RUnlock()

	// delete the below?
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

func (ds *Datastore) getTenantInstances(tenantID string) (instances map[string]*types.Instance, err error) {
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

// GetAllInstancesFromTenant will retrieve all instances belonging to a specific tenant
func (ds *Datastore) GetAllInstancesFromTenant(tenantID string) (instances []*types.Instance, err error) {
	ds.tenantsLock.RLock()

	t, ok := ds.tenants[tenantID]
	if ok {
		for _, val := range t.instances {
			instances = append(instances, val)
		}

		ds.tenantsLock.RUnlock()

		return
	}

	ds.tenantsLock.RUnlock()

	return nil, nil
}

// GetAllInstancesByNode will retrieve all the instances running on a specific compute Node.
func (ds *Datastore) GetAllInstancesByNode(nodeID string) (instances []*types.Instance, err error) {
	ds.nodesLock.RLock()
	n, ok := ds.nodes[nodeID]
	if ok {
		for _, val := range n.instances {
			instances = append(instances, val)
		}
		ds.nodesLock.RUnlock()
		return
	}
	ds.nodesLock.RUnlock()

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
	WHERE latest.node_id = ?
	`
	rows, err := tx.Query(query, nodeID)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

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

		instances = append(instances, i)
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

// GetInstanceFromTenant will be replaced soon with something else that makes more sense.
func (ds *Datastore) GetInstanceFromTenant(tenantID string, instanceID string) (*types.Instance, error) {
	var i types.Instance

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
	AND instances.id = ?
	`

	row := tx.QueryRow(query, tenantID, instanceID)

	var ID sql.NullString
	var tID sql.NullString
	var state sql.NullString
	var workloadID sql.NullString
	var nodeID sql.NullString
	var macAddress sql.NullString
	var ipAddress sql.NullString
	var sshIP sql.NullString
	var sshPort sql.NullInt64

	err = row.Scan(&ID, &tID, &state, &sshIP, &sshPort, &workloadID, &nodeID, &macAddress, &ipAddress)
	if err != nil {
		glog.V(2).Info("unable to retrieve instance %s from tenant %s (%s)", instanceID, tenantID, err)
		tx.Rollback()
		ds.tdbLock.RUnlock()
		return nil, err
	}

	tx.Commit()
	ds.tdbLock.RUnlock()

	if ID.Valid {
		i.Id = ID.String
	}
	if tID.Valid {
		i.TenantId = tID.String
	}
	if state.Valid {
		i.State = state.String
	}
	if workloadID.Valid {
		i.WorkloadId = workloadID.String
	}
	if nodeID.Valid {
		i.NodeId = nodeID.String
	}
	if macAddress.Valid {
		i.MACAddress = macAddress.String
	}
	if ipAddress.Valid {
		i.IPAddress = ipAddress.String
	}
	if sshIP.Valid {
		i.SSHIP = sshIP.String
	}
	if sshPort.Valid {
		i.SSHPort = int(sshPort.Int64)
	}

	return &i, err
}

// AddInstance will store a new instance in the datastore.
// The instance will be updated both in the cache and in the database
func (ds *Datastore) AddInstance(instance *types.Instance) (err error) {
	// add to cache
	ds.instancesLock.Lock()

	ds.instances[instance.Id] = instance

	instanceStat := payloads.CiaoServerStats{
		ID:        instance.Id,
		TenantID:  instance.TenantId,
		NodeID:    instance.NodeId,
		Timestamp: time.Now(),
		Status:    instance.State,
	}

	ds.instanceLastStatLock.Lock()
	ds.instanceLastStat[instance.Id] = instanceStat
	ds.instanceLastStatLock.Unlock()

	ds.instancesLock.Unlock()

	ds.tenantsLock.Lock()
	ds.tenants[instance.TenantId].instances[instance.Id] = instance
	ds.tenantsLock.Unlock()

	ds.dbLock.Lock()
	err = ds.create("instances", instance.Id, instance.TenantId, instance.WorkloadId, instance.MACAddress, instance.IPAddress)
	ds.dbLock.Unlock()

	return
}

// RestartFailure logs a RestartFailure in the datastore
func (ds *Datastore) RestartFailure(instanceID string, reason payloads.RestartFailureReason) (err error) {
	tenantID, err := ds.getInstanceOwner(instanceID)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("Restart Failure %s: %s", instanceID, reason.String())
	ds.logEvent(tenantID, string(userError), msg)
	return
}

// StopFailure logs a StopFailure in the datastore
func (ds *Datastore) StopFailure(instanceID string, reason payloads.StopFailureReason) (err error) {
	tenantID, err := ds.getInstanceOwner(instanceID)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("Stop Failure %s: %s", instanceID, reason.String())
	ds.logEvent(tenantID, string(userError), msg)
	return
}

// StartFailure will clean up after a failure to start an instance.
// If an instance was a CNCI, this function will remove the CNCI instance
// for this tenant. If the instance was a normal tenant instance, the
// IP address will be released and the instance will be deleted from the
// datastore.
func (ds *Datastore) StartFailure(instanceID string, reason payloads.StartFailureReason) (err error) {
	var tenantID string
	cnci, err := ds.isInstanceCNCI(instanceID)
	if err != nil {
		fmt.Println(err)
		return
	}
	if cnci == true {
		glog.Warning("CNCI ", instanceID, " Failed to start")
		tenantID, err = ds.getCNCITenant(instanceID)
		if err != nil {
			glog.Warning(err)
		}
		err = ds.removeTenantCNCI(tenantID, instanceID)
		if err != nil {
			glog.Warning(err)
		}
		msg := fmt.Sprintf("CNCI Start Failure %s: %s", instanceID, reason.String())
		ds.logEvent(tenantID, string(userError), msg)

		ds.cnciAddedLock.Lock()

		c, ok := ds.cnciAddedChans[tenantID]
		if ok {
			delete(ds.cnciAddedChans, tenantID)
		}

		ds.cnciAddedLock.Unlock()

		if c != nil {
			c <- false
		}

		return
	}

	tenantID, ipAddress, err := ds.getInstanceTenantNetwork(instanceID)
	if err != nil {
		return err
	}

	switch reason {
	case payloads.FullCloud,
		payloads.FullComputeNode,
		payloads.NoComputeNodes,
		payloads.NoNetworkNodes,
		payloads.InvalidPayload,
		payloads.InvalidData,
		payloads.ImageFailure,
		payloads.NetworkFailure:

		ds.instancesLock.Lock()
		i := ds.instances[instanceID]
		delete(ds.instances, instanceID)
		ds.instancesLock.Unlock()

		ds.tenantsLock.Lock()
		delete(ds.tenants[tenantID].instances, instanceID)
		ds.tenantsLock.Unlock()

		err = ds.deleteAllUsage(i, tenantID)
		if err != nil {
			glog.Warning(err)
		}

		go func() {
			cmd := `DELETE FROM instances WHERE id = '%s';`
			str := fmt.Sprintf(cmd, instanceID)
			ds.dbLock.Lock()
			_ = ds.exec(ds.getTableDB("instances"), str)
			ds.dbLock.Unlock()
		}()

		err = ds.ReleaseTenantIP(tenantID, ipAddress)
		if err != nil {
			glog.V(2).Info("StartFailure: ", err)
		}
	case payloads.LaunchFailure,
		payloads.AlreadyRunning,
		payloads.InstanceExists:
	}
	msg := fmt.Sprintf("Start Failure %s: %s", instanceID, reason.String())
	ds.logEvent(tenantID, string(userError), msg)
	return
}

// DeleteInstance removes an instance from the datastore.
func (ds *Datastore) DeleteInstance(instanceID string) (err error) {
	ds.instanceLastStatLock.Lock()
	delete(ds.instanceLastStat, instanceID)
	ds.instanceLastStatLock.Unlock()

	ds.instancesLock.Lock()
	i := ds.instances[instanceID]
	delete(ds.instances, instanceID)
	ds.instancesLock.Unlock()

	ds.tenantsLock.Lock()
	delete(ds.tenants[i.TenantId].instances, instanceID)
	ds.tenantsLock.Unlock()

	// we may not have received any node stats for this instance
	if i.NodeId != "" {
		ds.nodesLock.Lock()
		delete(ds.nodes[i.NodeId].instances, instanceID)
		ds.nodesLock.Unlock()
	}

	tenantID, ipAddress, err := ds.getInstanceTenantNetwork(instanceID)
	if err != nil {
		glog.V(2).Info("DeleteInstance: ", err)
		return
	}

	err = ds.deleteAllUsage(i, tenantID)
	if err != nil {
		glog.Warning(err)
	}

	cmd := `DELETE FROM instances WHERE id = '%s';`
	str := fmt.Sprintf(cmd, instanceID)
	ds.dbLock.Lock()
	err = ds.exec(ds.getTableDB("instances"), str)
	ds.dbLock.Unlock()
	if err != nil {
		glog.V(2).Info("DeleteInstance: ", err)
		return
	}

	err = ds.ReleaseTenantIP(tenantID, ipAddress)
	if err != nil {
		glog.V(2).Info("DeleteInstance: ", err)
		return
	}

	msg := fmt.Sprintf("Deleted Instance %s", instanceID)
	ds.logEvent(tenantID, string(userInfo), msg)
	return
}

// GetInstanceInfo will be replaced by something else soon that makes more sense.
func (ds *Datastore) GetInstanceInfo(instanceID string) (nodeID string, state string, err error) {
	// check cache of last stats first
	ds.instanceLastStatLock.RLock()
	instanceStat, ok := ds.instanceLastStat[instanceID]
	ds.instanceLastStatLock.RUnlock()

	if ok {
		return instanceStat.NodeID, instanceStat.Status, nil
	}

	datastore := ds.getTableDB("instances")

	query := `
	WITH latest AS
	(
		SELECT 	max(tdb.instance_statistics.timestamp),
			tdb.instance_statistics.instance_id,
			tdb.instance_statistics.state,
			tdb.instance_statistics.node_id
		FROM tdb.instance_statistics
		GROUP BY tdb.instance_statistics.instance_id
	)
	SELECT	latest.node_id,
		IFNULL(latest.state, "pending") AS state
	FROM instances
	LEFT JOIN latest
	ON instances.id = latest.instance_id
	WHERE instances.id = ?
	`

	var nullNodeID sql.NullString
	var nullState sql.NullString

	err = datastore.QueryRow(query, instanceID).Scan(&nullNodeID, &nullState)

	if nullNodeID.Valid {
		nodeID = nullNodeID.String
	}

	if nullState.Valid {
		state = nullState.String
	}

	return

}

func (ds *Datastore) getInstanceTenantNetwork(instanceID string) (tenantID string, ipAddress string, err error) {
	ds.instancesLock.RLock()
	i, ok := ds.instances[instanceID]
	ds.instancesLock.RUnlock()

	if ok {
		return i.TenantId, i.IPAddress, nil
	}

	datastore := ds.getTableDB("instances")

	err = datastore.QueryRow("SELECT tenant_id, ip FROM instances WHERE id = ?", instanceID).Scan(&tenantID, &ipAddress)
	return

}

func (ds *Datastore) getInstanceOwner(instanceID string) (tenantID string, err error) {
	ds.instancesLock.RLock()
	i, ok := ds.instances[instanceID]
	ds.instancesLock.RUnlock()

	if ok {
		return i.TenantId, nil
	}

	datastore := ds.getTableDB("instances")

	err = datastore.QueryRow("SELECT tenant_id FROM instances WHERE id = ?", instanceID).Scan(&tenantID)
	return

}

// AddUsage updates the accounting against this tenant's limits.
// usage is a map of resource name to the delta
func (ds *Datastore) AddUsage(tenantID string, instanceID string, usage map[string]int) (err error) {
	// update tenant cache
	ds.tenantsLock.Lock()
	tenant := ds.tenants[tenantID]
	if tenant != nil {
		for name, val := range usage {
			for i := range tenant.Resources {
				if tenant.Resources[i].Rname == name {
					tenant.Resources[i].Usage += val
					break
				}
			}
		}
		// increment instances count
		for i := range tenant.Resources {
			if tenant.Resources[i].Rtype == 1 {
				tenant.Resources[i].Usage++
				break
			}
		}
	}
	ds.tenantsLock.Unlock()

	go func(instanceID string, usage map[string]int) {
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
	}(instanceID, usage)
	return
}

func (ds *Datastore) deleteAllUsage(i *types.Instance, tenantID string) (err error) {
	// get the usage into a map
	ds.workloadsLock.RLock()
	wl := ds.workloads[i.WorkloadId]
	ds.workloadsLock.RUnlock()

	// convert RequestedResources into a map[string]int
	usage := make(map[string]int)
	for i := range wl.Defaults {
		usage[string(wl.Defaults[i].Type)] = wl.Defaults[i].Value
	}

	// update tenant usage in cache
	ds.tenantsLock.Lock()
	tenant := ds.tenants[tenantID]
	if tenant != nil {
		for name, val := range usage {
			for i := range tenant.Resources {
				if tenant.Resources[i].Rname == name {
					tenant.Resources[i].Usage -= val
					break
				}
			}
		}
		// decrement instances count
		for i := range tenant.Resources {
			if tenant.Resources[i].Rtype == 1 {
				tenant.Resources[i].Usage--
				break
			}
		}
	}
	ds.tenantsLock.Unlock()

	// update persistent store
	go func() {
		cmd := fmt.Sprintf("DELETE FROM usage WHERE instance_id = '%s';", i.Id)
		ds.dbLock.Lock()
		err = ds.exec(ds.getTableDB("usage"), cmd)
		ds.dbLock.Unlock()
	}()

	return
}

// HandleStats makes sure that the data from the stat payload is stored.
func (ds *Datastore) HandleStats(stat payloads.Stat) (err error) {
	if stat.Load != -1 {
		ds.addNodeStat(stat)
	}

	err = ds.addInstanceStats(stat.Instances, stat.NodeUUID)
	if err != nil {
		glog.Warning(err)
	}

	return
}

// HandleTraceReport stores the provided trace data in the datastore.
func (ds *Datastore) HandleTraceReport(trace payloads.Trace) (err error) {
	for index := range trace.Frames {
		i := trace.Frames[index]
		err = ds.addFrameStat(i)
		if err != nil {
			glog.Warning(err)
		}
	}
	return nil
}

// GetInstanceLastStats retrieves the last instances stats recieved for this node.
// It returns it in a format suitable for the compute API.
func (ds *Datastore) GetInstanceLastStats(nodeID string) payloads.CiaoServersStats {
	var serversStats payloads.CiaoServersStats

	ds.instanceLastStatLock.RLock()
	for _, instance := range ds.instanceLastStat {
		if instance.NodeID != nodeID {
			continue
		}
		serversStats.Servers = append(serversStats.Servers, instance)
	}
	ds.instanceLastStatLock.RUnlock()

	return serversStats
}

// GetNodeLastStats retrieves the last nodes stats recieved for this node.
// It returns it in a format suitable for the compute API.
func (ds *Datastore) GetNodeLastStats() payloads.CiaoComputeNodes {
	var computeNodes payloads.CiaoComputeNodes

	ds.nodeLastStatLock.RLock()
	for _, node := range ds.nodeLastStat {
		computeNodes.Nodes = append(computeNodes.Nodes, node)
	}
	ds.nodeLastStatLock.RUnlock()

	return computeNodes
}

func (ds *Datastore) addNodeStat(stat payloads.Stat) (err error) {
	ds.nodesLock.Lock()
	n, ok := ds.nodes[stat.NodeUUID]
	if !ok {
		n = &node{}
		n.instances = make(map[string]*types.Instance)
		ds.nodes[stat.NodeUUID] = n
	}
	n.ID = stat.NodeUUID
	n.Hostname = stat.NodeHostName
	ds.nodesLock.Unlock()

	cmd := `INSERT INTO node_statistics (node_id, mem_total_mb, mem_available_mb, disk_total_mb, disk_available_mb, load, cpus_online)
		VALUES('%s', %d, %d, %d, %d, %d, %d);`

	str := fmt.Sprintf(cmd, stat.NodeUUID, stat.MemTotalMB, stat.MemAvailableMB, stat.DiskTotalMB, stat.DiskAvailableMB, stat.Load, stat.CpusOnline)

	ds.tdbLock.Lock()

	err = ds.exec(ds.getTableDB("node_statistics"), str)

	ds.tdbLock.Unlock()

	cnStat := payloads.CiaoComputeNode{
		ID:            stat.NodeUUID,
		Status:        stat.Status,
		Load:          stat.Load,
		MemTotal:      stat.MemTotalMB,
		MemAvailable:  stat.MemAvailableMB,
		DiskTotal:     stat.DiskTotalMB,
		DiskAvailable: stat.DiskAvailableMB,
		OnlineCPUs:    stat.CpusOnline,
	}

	ds.nodeLastStatLock.Lock()

	delete(ds.nodeLastStat, stat.NodeUUID)
	ds.nodeLastStat[stat.NodeUUID] = cnStat

	ds.nodeLastStatLock.Unlock()

	return
}

var tenantUsagePeriodMinutes float64 = 5

func (ds *Datastore) updateTenantUsageNeeded(delta payloads.CiaoUsage, tenantID string) bool {
	if delta.VCPU == 0 &&
		delta.Memory == 0 &&
		delta.Disk == 0 {
		return false
	}

	return true
}

func (ds *Datastore) updateTenantUsage(delta payloads.CiaoUsage, tenantID string) {
	if ds.updateTenantUsageNeeded(delta, tenantID) == false {
		return
	}

	createNewUsage := true
	lastUsage := payloads.CiaoUsage{}

	ds.tenantUsageLock.Lock()

	tenantUsage := ds.tenantUsage[tenantID]
	if len(tenantUsage) != 0 {
		lastUsage = tenantUsage[len(tenantUsage)-1]
		// We will not create more than one entry per tenant every tenantUsagePeriodMinutes
		if time.Since(lastUsage.Timestamp).Minutes() < tenantUsagePeriodMinutes {
			createNewUsage = false
		}
	}

	newUsage := payloads.CiaoUsage{
		VCPU:   lastUsage.VCPU + delta.VCPU,
		Memory: lastUsage.Memory + delta.Memory,
		Disk:   lastUsage.Disk + delta.Disk,
	}

	// If we need to create a new usage entry, we timestamp it now.
	// If not we just update the last entry.
	if createNewUsage == true {
		newUsage.Timestamp = time.Now()
		ds.tenantUsage[tenantID] = append(ds.tenantUsage[tenantID], newUsage)
	} else {
		newUsage.Timestamp = lastUsage.Timestamp
		tenantUsage[len(tenantUsage)-1] = newUsage
	}

	ds.tenantUsageLock.Unlock()
}

func (ds *Datastore) GetTenantUsage(tenantID string, start time.Time, end time.Time) ([]payloads.CiaoUsage, error) {
	ds.tenantUsageLock.RLock()
	defer ds.tenantUsageLock.RUnlock()

	tenantUsage := ds.tenantUsage[tenantID]
	if tenantUsage == nil || len(tenantUsage) == 0 {
		return nil, fmt.Errorf("No usage history for %s", tenantID)
	}

	historyLength := len(tenantUsage)
	if tenantUsage[0].Timestamp.After(end) == true ||
		start.After(tenantUsage[historyLength-1].Timestamp) == true {
		return nil, nil
	}

	first := 0
	last := 0
	for _, u := range tenantUsage {
		if start.After(u.Timestamp) == true {
			first++
		}

		if end.After(u.Timestamp) == true {
			last++
		}
	}

	return tenantUsage[first:last], nil
}

func reduceToZero(v int) int {
	if v < 0 {
		return 0
	}

	return v
}

// BUG(kristen)
// we don't lock the tdb database yet.  Do we care enough about the
// temp db that we should protect it beyond just with transactions?
func (ds *Datastore) addInstanceStats(stats []payloads.InstanceStat, nodeID string) (err error) {
	ds.instancesLock.Lock()
	for index := range stats {
		stat := stats[index]
		instance, ok := ds.instances[stat.InstanceUUID]
		if ok {
			instance.State = stat.State
			instance.NodeId = nodeID
			instance.SSHIP = stat.SSHIP
			instance.SSHPort = stat.SSHPort
			ds.nodesLock.Lock()
			ds.nodes[nodeID].instances[instance.Id] = instance
			ds.nodesLock.Unlock()
		}
	}
	ds.instancesLock.Unlock()

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

		instanceStat := payloads.CiaoServerStats{
			ID:        stat.InstanceUUID,
			NodeID:    nodeID,
			Timestamp: time.Now(),
			Status:    stat.State,
			VCPUUsage: reduceToZero(stat.CPUUsage),
			MemUsage:  reduceToZero(stat.MemoryUsageMB),
			DiskUsage: reduceToZero(stat.DiskUsageMB),
		}

		ds.instanceLastStatLock.Lock()

		lastInstanceStat := ds.instanceLastStat[stat.InstanceUUID]

		deltaUsage := payloads.CiaoUsage{
			VCPU:   instanceStat.VCPUUsage - lastInstanceStat.VCPUUsage,
			Memory: instanceStat.MemUsage - lastInstanceStat.MemUsage,
			Disk:   instanceStat.DiskUsage - lastInstanceStat.DiskUsage,
		}

		go ds.updateTenantUsage(deltaUsage, lastInstanceStat.TenantID)

		instanceStat.TenantID = lastInstanceStat.TenantID

		delete(ds.instanceLastStat, stat.InstanceUUID)
		ds.instanceLastStat[stat.InstanceUUID] = instanceStat

		ds.instanceLastStatLock.Unlock()

	}
	tx.Commit()

	ds.tdbLock.Unlock()

	return
}

func (ds *Datastore) addFrameStat(stat payloads.FrameTrace) (err error) {
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
func (ds *Datastore) GetEventLog() (logEntries []*types.LogEntry, err error) {
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

// ClearNodeStats will delete all the log entries that were entered prior to the given time.
func (ds *Datastore) ClearNodeStats(before time.Time) (err error) {
	ds.tdbLock.Lock()
	cmd := "DELETE FROM node_statistics WHERE timestamp < '%s'"
	str := fmt.Sprintf(cmd, before)
	err = ds.exec(ds.getTableDB("node_statistics"), str)
	ds.tdbLock.Unlock()
	return
}

// GetNodeStats returns all node stats received between start and end time.
func (ds *Datastore) GetNodeStats(start time.Time, end time.Time) (statsRows []*types.NodeStats, err error) {
	datastore := ds.getTableDB("node_statistics")
	ds.tdbLock.RLock()

	rows, err := datastore.Query("SELECT timestamp, node_id, load, mem_total_mb, mem_available_mb, disk_total_mb, disk_available_mb, cpus_online FROM node_statistics WHERE timestamp BETWEEN ? AND ?", start, end)
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r types.NodeStats

		err = rows.Scan(&r.Timestamp, &r.NodeId, &r.Load, &r.MemTotalMB, &r.MemAvailableMB, &r.DiskTotalMB, &r.DiskAvailableMB, &r.CpusOnline)
		if err != nil {
			ds.tdbLock.RUnlock()
			return nil, err
		}
		statsRows = append(statsRows, &r)
	}

	if len(statsRows) == 0 {
		ds.tdbLock.RUnlock()
		return nil, err
	}
	ds.tdbLock.RUnlock()

	return statsRows, err
}

// GetNodeSummary provides a summary the state and count of instances running per node.
func (ds *Datastore) GetNodeSummary() (Summary []*types.NodeSummary, err error) {
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

// GetTenantCNCISummary retrieves information about a given CNCI id, or all CNCIs
// If the cnci string is the null string, then this function will retrieve all
// tenants.  If cnci is not null, it will only provide information about a specific
// cnci.
func (ds *Datastore) GetTenantCNCISummary(cnci string) (cncis []types.TenantCNCI, err error) {
	datastore := ds.getTableDB("tenants")
	var query string

	if cnci == "" {
		query = "SELECT id, cnci_ip, cnci_mac, cnci_id FROM tenants"
	} else {
		query = fmt.Sprintf("SELECT id, cnci_ip, cnci_mac, cnci_id FROM tenants WHERE cnci_id = '%s'", cnci)
	}
	rows, err := datastore.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subnetBytes := []byte{0, 0}
	cncis = make([]types.TenantCNCI, 0)
	for rows.Next() {
		var cn types.TenantCNCI
		err = rows.Scan(&cn.TenantID, &cn.IPAddress, &cn.MACAddress, &cn.InstanceID)
		if err != nil {
			return
		}

		tenant, err := ds.getTenant(cn.TenantID)
		if err != nil && tenant != nil {
			continue
		}

		for _, subnet := range tenant.subnets {
			binary.BigEndian.PutUint16(subnetBytes, (uint16)(subnet))
			cn.Subnets = append(cn.Subnets, fmt.Sprintf("Subnet 172.%d.%d.0/8", subnetBytes[0], subnetBytes[1]))
		}

		cncis = append(cncis, cn)
	}

	return cncis, err
}

// GetFrameStatistics will return trace data by label id.
func (ds *Datastore) GetFrameStatistics(label string) (stats []types.FrameStat, err error) {
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
func (ds *Datastore) GetBatchFrameSummary() (stats []types.BatchFrameSummary, err error) {
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
func (ds *Datastore) GetBatchFrameStatistics(label string) (stats []types.BatchFrameStat, err error) {
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

// GetNodes will retrieve a list of all the nodes that we have information on
func (ds *Datastore) GetNodes() (nodes []types.Node, err error) {
	ds.nodesLock.RLock()

	for i := range ds.nodes {
		nodes = append(nodes, ds.nodes[i].Node)
	}

	ds.nodesLock.RUnlock()

	return nodes, nil
}
