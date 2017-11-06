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
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/payloads"
	"github.com/golang/glog"
	sqlite3 "github.com/mattn/go-sqlite3"

	"github.com/pkg/errors"
)

type sqliteDB struct {
	db            *sql.DB
	dbName        string
	tables        []persistentData
	workloadsPath string
	dbLock        *sync.Mutex
}

type persistentData interface {
	Init() error
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

func (d namedData) Name() (name string) {
	return d.name
}

func (d namedData) DB() *sql.DB {
	return d.db
}

type logData struct {
	namedData
}

func (d logData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS log
		(
		id integer primary key,
		tenant_id varchar(32),
		node_id varchar(32),
		type string,
		message string,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
		);`

	return d.ds.exec(d.db, cmd)
}

type subnetData struct {
	namedData
}

func (d subnetData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS tenant_network
		(
		tenant_id varchar(32),
		subnet unsigned int,
		rest unsigned int,
		foreign key(tenant_id) references tenants(id)
		);`

	return d.ds.exec(d.db, cmd)
}

// Handling of Instance specific data
type instanceData struct {
	namedData
}

func (d instanceData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS instances
		(
		id string primary key,
		tenant_id string,
		workload_id string,
		mac_address string,
		vnic_uuid string,
		subnet string,
		ip string,
		create_time DATETIME,
		name string,
		cnci int,
		foreign key(tenant_id) references tenants(id),
		foreign key(workload_id) references workload_template(id),
		unique(tenant_id, ip, mac_address)
		);`

	return d.ds.exec(d.db, cmd)
}

// Volume Data
type blockData struct {
	namedData
}

func (d blockData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS block_data
		(
		id string primary_key,
		tenant_id string,
		size integer,
		state string,
		create_time DATETIME,
		name string,
		description string,
		internal int,
		foreign key(tenant_id) references tenants(id)
		);`

	return d.ds.exec(d.db, cmd)
}

type attachments struct {
	namedData
}

func (d attachments) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS attachments
		(
		id string primary key,
		instance_id string,
		block_id string,
		ephemeral int,
		boot int,
		foreign key(instance_id) references instances(id),
		foreign key(block_id) references block_data(id)
		);`

	return d.ds.exec(d.db, cmd)
}

// workload storage resources

type workloadStorage struct {
	namedData
}

func (d workloadStorage) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS workload_storage
	        (
		workload_id string,
		volume_id string,
		bootable int,
		ephemeral int,
		size integer,
		source_type string,
		source_id string,
		tag string,
		foreign key(workload_id) references workloads(id),
		foreign key(volume_id) references block_data(id)
		);`

	return d.ds.exec(d.db, cmd)
}

// Tenants data
type tenantData struct {
	namedData
}

func (d tenantData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS tenants
		(
		id varchar(32) primary key,
		name text,
		subnet_bits int
		);`

	return d.ds.exec(d.db, cmd)
}

// workload resources
type workloadResourceData struct {
	namedData
}

func (d workloadResourceData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS workload_resources
		(
		workload_id varchar(32),
		resource_type string,
		default_value int,
		estimated_value int,
		mandatory int,
		foreign key(workload_id) references workload_template(id)
		);
		CREATE UNIQUE INDEX IF NOT EXISTS wlr_index
		ON workload_resources(workload_id, resource_type);`

	return d.ds.exec(d.db, cmd)
}

// workload template data
type workloadTemplateData struct {
	namedData
}

func (d workloadTemplateData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS workload_template
		(
		id varchar(32) primary key,
		tenant_id varchar(32),
		description text,
		filename text,
		fw_type text,
		vm_type text,
		image_name text,
		visibility text
		);`

	return d.ds.exec(d.db, cmd)
}

// statistics
type nodeStatisticsData struct {
	namedData
}

func (d nodeStatisticsData) Init() error {
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

	return d.ds.exec(d.db, cmd)
}

type instanceStatisticsData struct {
	namedData
}

func (d instanceStatisticsData) Init() error {
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

	return d.ds.exec(d.db, cmd)
}

type frameStatisticsData struct {
	namedData
}

func (d frameStatisticsData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS frame_statistics
		(
			id integer primary key autoincrement not null,
			label string,
			type string,
			operand string,
			start_timestamp DATETIME,
			end_timestamp DATETIME
		);`

	return d.ds.exec(d.db, cmd)
}

type traceData struct {
	namedData
}

func (d traceData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS trace_data
		(
			id integer primary key autoincrement not null,
			frame_id int,
			ssntp_uuid varchar(32),
			tx_timestamp DATETIME,
			rx_timestamp DATETIME,
			foreign key(frame_id) references frame_statistics(id)
		);`

	return d.ds.exec(d.db, cmd)
}

type poolData struct {
	namedData
}

func (d poolData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS pools
		(
			id varchar(32),
			name string,
			free int,
			total int,
			PRIMARY KEY(id, name)
		);`

	return d.ds.exec(d.db, cmd)
}

type subnetPoolData struct {
	namedData
}

func (d subnetPoolData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS subnet_pool
		(
			id varchar(32) primary key,
			pool_id varchar(32),
			cidr string
		);`

	return d.ds.exec(d.db, cmd)
}

type addressData struct {
	namedData
}

func (d addressData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS address_pool
		(
			id varchar(32) primary key,
			pool_id varchar(32),
			address string
		);`

	return d.ds.exec(d.db, cmd)
}

type mappedIPData struct {
	namedData
}

func (d mappedIPData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS mapped_ips
		(
			id varchar(32) primary key,
			external_ip string,
			instance_id varchar(32),
			pool_id varchar(32)
		);`

	return d.ds.exec(d.db, cmd)
}

type quotaData struct {
	namedData
}

func (d quotaData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS quotas
		(
			tenant_id string,
			name string,
			value int,
			unique(tenant_id, name)
		);`

	return d.ds.exec(d.db, cmd)
}

type imageData struct {
	namedData
}

func (d imageData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS images
		(
			id varchar(32) primary key,
			state string,		
			tenant_id string,
			name string,
			createtime DATETIME,
			size int,
			visibility string
		);`

	return d.ds.exec(d.db, cmd)
}

func (ds *sqliteDB) exec(db *sql.DB, cmd string) error {
	glog.V(2).Info("exec: ", cmd)

	_, err := db.Exec(cmd)

	return err
}

// This function is deprecated and will be removed soon. It should not be used
// for newly written or updated code.
func (ds *sqliteDB) create(tableName string, record ...interface{}) error {
	// get database location of this table
	db := ds.getTableDB(tableName)

	if db == nil {
		return errors.New("Bad table name")
	}

	var values []string
	for _, val := range record {
		v := reflect.ValueOf(val)

		var newval string

		// enclose strings in quotes to not confuse sqlite
		if v.Kind() == reflect.String {
			newval = fmt.Sprintf("'%v'", val)
		} else if v.Kind() == reflect.Bool {
			if val.(bool) {
				newval = "1"
			} else {
				newval = "0"
			}
		} else {
			newval = fmt.Sprintf("%v", val)
		}

		values = append(values, newval)
	}

	args := strings.Join(values, ",")
	cmd := "INSERT into " + tableName + " VALUES (" + args + ");"

	return ds.exec(db, cmd)
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

// init initializes the private data for the database object.
// The datastore caches are also filled.
func (ds *sqliteDB) init(config Config) error {
	u, err := url.Parse(config.PersistentURI)
	if err != nil {
		return fmt.Errorf("Invalid URL (%s) for persistent data store: %v", config.PersistentURI, err)
	}

	if u.Scheme == "file" {
		dbDir := filepath.Dir(u.Path)
		err = os.MkdirAll(dbDir, 0755)
		if err != nil && dbDir != "." {
			return fmt.Errorf("Unable to create db directory (%s) %v", dbDir, err)
		}
	}

	err = ds.Connect(config.PersistentURI)
	if err != nil {
		return err
	}

	ds.dbLock = &sync.Mutex{}

	ds.tables = []persistentData{
		tenantData{namedData{ds: ds, name: "tenants", db: ds.db}},
		instanceData{namedData{ds: ds, name: "instances", db: ds.db}},
		workloadTemplateData{namedData{ds: ds, name: "workload_template", db: ds.db}},
		workloadResourceData{namedData{ds: ds, name: "workload_resources", db: ds.db}},
		nodeStatisticsData{namedData{ds: ds, name: "node_statistics", db: ds.db}},
		logData{namedData{ds: ds, name: "log", db: ds.db}},
		subnetData{namedData{ds: ds, name: "tenant_network", db: ds.db}},
		instanceStatisticsData{namedData{ds: ds, name: "instance_statistics", db: ds.db}},
		frameStatisticsData{namedData{ds: ds, name: "frame_statistics", db: ds.db}},
		traceData{namedData{ds: ds, name: "trace_data", db: ds.db}},
		blockData{namedData{ds: ds, name: "block_data", db: ds.db}},
		attachments{namedData{ds: ds, name: "attachments", db: ds.db}},
		workloadStorage{namedData{ds: ds, name: "workload_storage", db: ds.db}},
		poolData{namedData{ds: ds, name: "pools", db: ds.db}},
		subnetPoolData{namedData{ds: ds, name: "subnet_pool", db: ds.db}},
		addressData{namedData{ds: ds, name: "address_pool", db: ds.db}},
		mappedIPData{namedData{ds: ds, name: "mapped_ips", db: ds.db}},
		quotaData{namedData{ds: ds, name: "quotas", db: ds.db}},
		imageData{namedData{ds: ds, name: "images", db: ds.db}},
	}

	ds.workloadsPath = config.InitWorkloadsPath
	if err := os.MkdirAll(ds.workloadsPath, 0755); err != nil {
		return errors.Wrap(err, "Error creating workload directory")
	}

	for _, table := range ds.tables {
		err = table.Init()
		if err != nil {
			return err
		}
	}

	return nil
}

var pSQLLiteConfig = []string{
	"PRAGMA page_size = 32768",
	"PRAGMA synchronous = OFF",
	"PRAGMA temp_store = MEMORY",
	"PRAGMA busy_timeout = 1000",
	"PRAGMA journal_mode=WAL",
}

func (ds *sqliteDB) sqliteConnect(name string, URI string, config []string) (*sql.DB, error) {
	db, err := sql.Open(name, URI)
	if err != nil {
		return nil, err
	}

	for i := range config {
		_, err = db.Exec(config[i])
		if err != nil {
			glog.Warning(err)
		}
	}

	err = db.Ping()
	if err != nil {
		glog.Warning(err)
		return nil, err
	}

	return db, nil
}

func (ds *sqliteDB) Connect(persistentURI string) error {
	sql.Register(persistentURI, &sqlite3.SQLiteDriver{})

	db, err := ds.sqliteConnect(persistentURI, persistentURI, pSQLLiteConfig)
	if err != nil {
		return err
	}

	ds.db = db
	ds.dbName = persistentURI

	return err
}

// Disconnect is used to close the connection to the sql database
func (ds *sqliteDB) disconnect() {
	_ = ds.db.Close()
}

func (ds *sqliteDB) logEvent(event types.LogEntry) error {
	db := ds.getTableDB("log")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("INSERT INTO log (tenant_id, node_id, type, message) VALUES (?, ?, ?, ?)", event.TenantID, event.NodeID, event.EventType, event.Message)

	return err
}

// ClearLog will remove all the event entries from the event log
func (ds *sqliteDB) clearLog() error {
	db := ds.getTableDB("log")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("DELETE FROM log")

	return err
}

func (ds *sqliteDB) getConfig(ID string) (string, error) {
	var configFile string

	db := ds.getTableDB("workload_template")

	err := db.QueryRow("SELECT filename FROM workload_template where id = ?", ID).Scan(&configFile)

	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/%s", ds.workloadsPath, configFile)
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	config := string(bytes)

	return config, nil
}

func (ds *sqliteDB) getWorkloadDefaults(ID string) ([]payloads.RequestedResource, error) {
	query := `SELECT resource_type, default_value, mandatory FROM workload_resources
	     WHERE workload_id = ? ORDER BY resource_type `

	db := ds.getTableDB("workload_resources")

	rows, err := db.Query(query, ID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var defaults []payloads.RequestedResource

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

	return defaults, nil
}

// lock must be held by caller
func (ds *sqliteDB) createWorkloadDefault(tx *sql.Tx, workloadID string, resource payloads.RequestedResource) error {
	_, err := tx.Exec("INSERT INTO workload_resources (workload_id, resource_type, default_value, estimated_value, mandatory) VALUES (?, ?, ?, ?, ?)", workloadID, string(resource.Type), resource.Value, resource.Value, resource.Mandatory)

	return err
}

// lock must be held by caller
func (ds *sqliteDB) deleteWorkloadDefault(tx *sql.Tx, workloadID string) error {
	_, err := tx.Exec("DELETE FROM workload_resources WHERE workload_id = ?", workloadID)

	return err
}

// lock must be held by caller
func (ds *sqliteDB) createWorkloadStorage(tx *sql.Tx, workloadID string, storage *types.StorageResource) error {
	_, err := tx.Exec("INSERT INTO workload_storage (workload_id, volume_id, bootable, ephemeral, size, source_type, source_id, tag) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", workloadID, storage.ID, storage.Bootable, storage.Ephemeral, storage.Size, string(storage.SourceType), storage.SourceID, storage.Tag)

	return err
}

// lock must be held by caller
func (ds *sqliteDB) deleteWorkloadStorage(tx *sql.Tx, workloadID string) error {
	_, err := tx.Exec("DELETE FROM workload_storage WHERE workload_id = ?", workloadID)

	return err
}

func (ds *sqliteDB) getWorkloadStorage(ID string) ([]types.StorageResource, error) {
	query := `SELECT volume_id, bootable, ephemeral, size,
			 source_type, source_id, tag
		  FROM 	workload_storage
		  WHERE workload_id = ?`

	rows, err := ds.db.Query(query, ID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	res := []types.StorageResource{}
	var sourceType string

	for rows.Next() {
		var r types.StorageResource
		err := rows.Scan(&r.ID, &r.Bootable, &r.Ephemeral, &r.Size, &sourceType, &r.SourceID, &r.Tag)

		if err != nil {
			return []types.StorageResource{}, err
		}
		r.SourceType = types.SourceType(sourceType)
		res = append(res, r)
	}
	return res, nil
}

func (ds *sqliteDB) addTenant(ID string, config types.TenantConfig) error {
	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	err := ds.create("tenants", ID, config.Name, config.SubnetBits)

	return err
}

func (ds *sqliteDB) getTenant(ID string) (*tenant, error) {
	query := `SELECT	tenants.id,
				tenants.name,
				tenants.subnet_bits
		  FROM tenants
		  WHERE tenants.id = ?`

	db := ds.db

	row := db.QueryRow(query, ID)

	t := &tenant{}

	err := row.Scan(&t.ID, &t.Name, &t.SubnetBits)
	if err != nil {
		glog.Warning("unable to retrieve tenant from tenants")

		if err == sql.ErrNoRows {
			// not an error, it's just not there.
			err = nil
		}

		return nil, err
	}

	// for these items below, its ok to get err returned
	// because a tenant could simply not have used any
	// resources or networks yet.
	err = ds.getTenantNetwork(t)
	if err != nil {
		glog.V(2).Info(err)
	}

	t.instances, err = ds.getTenantInstances(t.ID)
	if err != nil {
		glog.V(2).Info(err)
	}

	t.devices, err = ds.getTenantDevices(t.ID)
	if err != nil {
		glog.V(2).Info(err)
	}

	return t, err
}

func (ds *sqliteDB) getWorkloads() ([]types.Workload, error) {
	var workloads []types.Workload

	db := ds.db

	query := `SELECT id,
			 tenant_id,
			 description,
			 fw_type,
			 vm_type,
			 image_name,
			 visibility
		  FROM workload_template`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var wl types.Workload

		var VMType string
		var visibility string

		err = rows.Scan(&wl.ID, &wl.TenantID, &wl.Description, &wl.FWType, &VMType, &wl.ImageName, &visibility)
		if err != nil {
			return nil, err
		}

		wl.Visibility = types.Visibility(visibility)

		if wl.Visibility == types.Internal {
			continue
		}

		wl.Config, err = ds.getConfig(wl.ID)
		if err != nil {
			return nil, err
		}

		wl.Defaults, err = ds.getWorkloadDefaults(wl.ID)
		if err != nil {
			return nil, err
		}

		wl.Storage, err = ds.getWorkloadStorage(wl.ID)
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

func (ds *sqliteDB) addWorkload(w types.Workload) error {
	db := ds.getTableDB("workload_template")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// add in workload resources
	for _, d := range w.Defaults {
		err := ds.createWorkloadDefault(tx, w.ID, d)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	// add in any workload storage resources
	if len(w.Storage) > 0 {
		for i := range w.Storage {
			err := ds.createWorkloadStorage(tx, w.ID, &w.Storage[i])
			if err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}

	// write config to file.
	filename := fmt.Sprintf("%s_config.yaml", w.ID)
	path := fmt.Sprintf("%s/%s", ds.workloadsPath, filename)
	err = ioutil.WriteFile(path, []byte(w.Config), 0644)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec("INSERT INTO workload_template (id, tenant_id, description, filename, fw_type, vm_type, image_name, visibility) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", w.ID, w.TenantID, w.Description, filename, w.FWType, string(w.VMType), w.ImageName, w.Visibility)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = tx.Commit()
	return err
}

func (ds *sqliteDB) deleteWorkload(ID string) error {
	db := ds.getTableDB("workload_template")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	err = ds.deleteWorkloadDefault(tx, ID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = ds.deleteWorkloadStorage(tx, ID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec("DELETE FROM workload_template WHERE id = ?", ID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	filename := fmt.Sprintf("%s_config.yaml", ID)
	path := filepath.Join(ds.workloadsPath, filename)
	err = os.Remove(path)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = tx.Commit()
	return err
}

func (ds *sqliteDB) getTenants() ([]*tenant, error) {
	var tenants []*tenant

	db := ds.getTableDB("tenants")

	query := `SELECT	tenants.id,
				tenants.name,
				tenants.subnet_bits
		  FROM tenants `

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id sql.NullString
		var name sql.NullString

		t := new(tenant)
		err = rows.Scan(&id, &name, &t.SubnetBits)
		if err != nil {
			return nil, err
		}

		if id.Valid {
			t.ID = id.String
		}

		if name.Valid {
			t.Name = name.String
		}

		err = ds.getTenantNetwork(t)
		if err != nil {
			return nil, err
		}

		t.instances, err = ds.getTenantInstances(t.ID)
		if err != nil {
			return nil, err
		}

		t.devices, err = ds.getTenantDevices(t.ID)
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

func (ds *sqliteDB) claimTenantIP(tenantID string, subnetInt uint32, rest uint32) error {
	db := ds.getTableDB("tenant_network")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("INSERT INTO tenant_network VALUES(?, ?, ?)", tenantID, subnetInt, rest)

	return err
}

func (ds *sqliteDB) claimTenantIPs(tenantID string, IPs []tenantIP) error {
	db := ds.getTableDB("tenant_network")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	cmd := `INSERT INTO tenant_network VALUES(?, ?, ?)`

	stmt, err := tx.Prepare(cmd)
	if err != nil {
		tx.Rollback()
		return err
	}

	defer stmt.Close()

	for _, ip := range IPs {
		_, err = stmt.Exec(tenantID, ip.subnet, ip.host)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (ds *sqliteDB) releaseTenantIP(tenantID string, subnetInt uint32, rest uint32) error {
	db := ds.getTableDB("tenant_network")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("DELETE FROM tenant_network WHERE tenant_id = ? AND subnet = ? AND rest = ?", tenantID, subnetInt, rest)

	return err
}

func (ds *sqliteDB) getTenantNetwork(tenant *tenant) error {
	tenant.network = make(map[uint32]map[uint32]bool)

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	db := ds.getTableDB("tenant_network")

	// get all subnet,rest values for this tenant
	query := `SELECT subnet, rest
		  FROM tenant_network
		  WHERE tenant_id = ?`

	rows, err := db.Query(query, tenant.ID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var subnetInt uint32
		var rest uint32

		err = rows.Scan(&subnetInt, &rest)
		if err != nil {
			return err
		}

		_, ok := tenant.network[subnetInt]
		if !ok {
			sub := make(map[uint32]bool)
			tenant.network[subnetInt] = sub
		}

		tenant.network[subnetInt][rest] = true
	}

	return err
}

func (ds *sqliteDB) updateTenant(tenant *types.Tenant) error {
	db := ds.getTableDB("tenants")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("UPDATE tenants SET name = ?, subnet_bits = ? WHERE id = ?", tenant.Name, tenant.SubnetBits, tenant.ID)

	return err
}

func (ds *sqliteDB) deleteTenant(tenantID string) error {
	db := ds.getTableDB("tenants")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// first delete any quotas associated with this tenant
	_, err = tx.Exec("DELETE FROM quotas WHERE tenant_id = ?", tenantID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = tx.Commit()

	return err
}

func (ds *sqliteDB) getInstances() ([]*types.Instance, error) {
	var instances []*types.Instance

	db := ds.getTableDB("instances")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	query := `
	WITH latest AS
	(
		SELECT 	max(instance_statistics.timestamp),
			instance_statistics.instance_id,
			instance_statistics.state,
			instance_statistics.ssh_ip,
			instance_statistics.ssh_port,
			instance_statistics.node_id
		FROM instance_statistics
		GROUP BY instance_statistics.instance_id
	)
	SELECT	instances.id,
		instances.tenant_id,
		IFNULL(latest.state, "` + payloads.ComputeStatusPending + `") AS state,
		workload_id,
		IFNULL(latest.ssh_ip, "Not Assigned") as ssh_ip,
		latest.ssh_port as ssh_port,
		IFNULL(latest.node_id, "Not Assigned") as node_id,
		mac_address,
		vnic_uuid,
		subnet,
		ip,
		name,
		cnci
	FROM instances
	LEFT JOIN latest
	ON instances.id = latest.instance_id
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var i types.Instance

		var sshPort sql.NullInt64

		err = rows.Scan(&i.ID, &i.TenantID, &i.State, &i.WorkloadID, &i.SSHIP, &sshPort, &i.NodeID, &i.MACAddress, &i.VnicUUID, &i.Subnet, &i.IPAddress, &i.Name, &i.CNCI)
		if err != nil {
			return nil, err
		}

		if sshPort.Valid {
			i.SSHPort = int(sshPort.Int64)
		}

		i.StateChange = sync.NewCond(&sync.Mutex{})

		instances = append(instances, &i)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return instances, nil
}

func (ds *sqliteDB) getTenantInstances(tenantID string) (map[string]*types.Instance, error) {
	db := ds.getTableDB("instances")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	query := `
	WITH latest AS
	(
		SELECT 	max(instance_statistics.timestamp),
			instance_statistics.instance_id,
			instance_statistics.state,
			instance_statistics.ssh_ip,
			instance_statistics.ssh_port,
			instance_statistics.node_id
		FROM instance_statistics
		GROUP BY instance_statistics.instance_id
	)
	SELECT	instances.id,
		instances.tenant_id,
		IFNULL(latest.state, "` + payloads.ComputeStatusPending + `") AS state,
		IFNULL(latest.ssh_ip, "Not Assigned") AS ssh_ip,
		latest.ssh_port AS ssh_port,
		workload_id,
		latest.node_id,
		mac_address,
		vnic_uuid,
		subnet,
		ip,
		name,
		cnci
	FROM instances
	LEFT JOIN latest
	ON instances.id = latest.instance_id
	WHERE instances.tenant_id = ?
	`

	rows, err := db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	instances := make(map[string]*types.Instance)
	for rows.Next() {
		var nodeID sql.NullString
		var sshIP sql.NullString
		var sshPort sql.NullInt64

		i := &types.Instance{}

		err = rows.Scan(&i.ID, &i.TenantID, &i.State, &sshIP, &sshPort, &i.WorkloadID, &nodeID, &i.MACAddress, &i.VnicUUID, &i.Subnet, &i.IPAddress, &i.Name, &i.CNCI)
		if err != nil {
			return nil, err
		}

		if nodeID.Valid {
			i.NodeID = nodeID.String
		}

		if sshIP.Valid {
			i.SSHIP = sshIP.String
		}

		if sshPort.Valid {
			i.SSHPort = int(sshPort.Int64)
		}

		i.StateChange = sync.NewCond(&sync.Mutex{})

		instances[i.ID] = i
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return instances, nil
}

func (ds *sqliteDB) addInstance(instance *types.Instance) error {
	db := ds.getTableDB("instances")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("INSERT INTO instances VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", instance.ID, instance.TenantID, instance.WorkloadID, instance.MACAddress, instance.VnicUUID, instance.Subnet, instance.IPAddress, instance.CreateTime.Format(time.RFC3339Nano), instance.Name, instance.CNCI)

	return err
}

func (ds *sqliteDB) deleteInstance(instanceID string) error {
	db := ds.getTableDB("instances")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("DELETE FROM instances WHERE id = ?", instanceID)

	return err
}

func (ds *sqliteDB) updateInstance(instance *types.Instance) error {
	db := ds.getTableDB("instances")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("UPDATE instances SET mac_address = ?, ip = ? WHERE id = ?", instance.MACAddress, instance.IPAddress, instance.ID)

	return err
}

func (ds *sqliteDB) addNodeStat(stat payloads.Stat) error {
	db := ds.getTableDB("node_statistics")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("INSERT INTO node_statistics (node_id, mem_total_mb, mem_available_mb, disk_total_mb, disk_available_mb, load, cpus_online) VALUES(?, ?, ?, ?, ?, ?, ?)", stat.NodeUUID, stat.MemTotalMB, stat.MemAvailableMB, stat.DiskTotalMB, stat.DiskAvailableMB, stat.Load, stat.CpusOnline)

	return err
}

func (ds *sqliteDB) addInstanceStats(stats []payloads.InstanceStat, nodeID string) error {
	db := ds.getTableDB("instance_statistics")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	cmd := `INSERT INTO instance_statistics (instance_id, memory_usage_mb, disk_usage_mb, cpu_usage, state, node_id, ssh_ip, ssh_port)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(cmd)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	defer func() { _ = stmt.Close() }()

	for index := range stats {
		stat := stats[index]

		_, err = stmt.Exec(stat.InstanceUUID, stat.MemoryUsageMB, stat.DiskUsageMB, stat.CPUUsage, stat.State, nodeID, stat.SSHIP, stat.SSHPort)
		if err != nil {
			glog.Warning(err)
			// but keep going
		}
	}

	err = tx.Commit()

	return err
}

func (ds *sqliteDB) addFrameStat(stat payloads.FrameTrace) error {
	db := ds.getTableDB("frame_statistics")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	query := `INSERT INTO frame_statistics (label, type, operand, start_timestamp, end_timestamp)
		  VALUES(?, ?, ?, ?, ?)`

	_, err = tx.Exec(query, stat.Label, stat.Type, stat.Operand, stat.StartTimestamp, stat.EndTimestamp)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	var id int

	err = tx.QueryRow("SELECT last_insert_rowid();").Scan(&id)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	for index := range stat.Nodes {
		t := stat.Nodes[index]

		cmd := `INSERT INTO trace_data (frame_id, ssntp_uuid, tx_timestamp, rx_timestamp)
			VALUES(?, ?, ?, ?);`

		_, err = tx.Exec(cmd, id, t.SSNTPUUID, t.TxTimestamp, t.RxTimestamp)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	err = tx.Commit()

	return err
}

// GetEventLog retrieves all the log entries stored in the datastore.
func (ds *sqliteDB) getEventLog() ([]*types.LogEntry, error) {
	var logEntries []*types.LogEntry

	db := ds.getTableDB("log")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	rows, err := db.Query("SELECT timestamp, tenant_id, node_id, type, message FROM log")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	logEntries = make([]*types.LogEntry, 0)
	for rows.Next() {
		var e types.LogEntry
		err = rows.Scan(&e.Timestamp, &e.TenantID, &e.NodeID, &e.EventType, &e.Message)
		if err != nil {
			return nil, err
		}
		logEntries = append(logEntries, &e)
	}

	return logEntries, err
}

// GetBatchFrameSummary will retieve the count of traces we have for a specific label
func (ds *sqliteDB) getBatchFrameSummary() ([]types.BatchFrameSummary, error) {
	var stats []types.BatchFrameSummary

	db := ds.getTableDB("frame_statistics")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	query := `SELECT label, count(id)
		  FROM frame_statistics
		  GROUP BY label;`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	stats = make([]types.BatchFrameSummary, 0)

	for rows.Next() {
		var stat types.BatchFrameSummary

		err = rows.Scan(&stat.BatchID, &stat.NumInstances)
		if err != nil {
			return nil, err
		}

		stats = append(stats, stat)
	}

	return stats, err
}

// GetBatchFrameStatistics will show individual trace data per instance for a batch of trace data.
// The batch is identified by the label.
func (ds *sqliteDB) getBatchFrameStatistics(label string) ([]types.BatchFrameStat, error) {
	var stats []types.BatchFrameStat

	db := ds.getTableDB("frame_statistics")

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

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	rows, err := db.Query(query, label)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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
			return nil, err
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

	return stats, err
}

func (ds *sqliteDB) getTenantDevices(tenantID string) (map[string]types.Volume, error) {
	devices := make(map[string]types.Volume)

	db := ds.getTableDB("block_data")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	query := `SELECT	block_data.id,
				block_data.tenant_id,
				block_data.size,
				block_data.state,
				block_data.create_time,
				block_data.name,
				block_data.description,
				block_data.internal
		  FROM	block_data
		  WHERE block_data.tenant_id = ?`

	rows, err := db.Query(query, tenantID)
	if err != nil {
		return devices, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var state string
		var data types.Volume

		err = rows.Scan(&data.ID, &data.TenantID, &data.Size, &state, &data.CreateTime, &data.Name, &data.Description, &data.Internal)
		if err != nil {
			continue
		}

		data.State = types.BlockState(state)
		devices[data.ID] = data
	}

	if err = rows.Err(); err != nil {
		return devices, err
	}

	return devices, nil
}

func (ds *sqliteDB) getAllBlockData() (map[string]types.Volume, error) {
	devices := make(map[string]types.Volume)

	db := ds.getTableDB("block_data")

	query := `SELECT	block_data.id,
				block_data.tenant_id,
				block_data.size,
				block_data.state,
				block_data.create_time,
				block_data.name,
				block_data.description,
				block_data.internal
		  FROM	block_data `

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var data types.Volume
		var state string

		err = rows.Scan(&data.ID, &data.TenantID, &data.Size, &state, &data.CreateTime, &data.Name, &data.Description, &data.Internal)
		if err != nil {
			continue
		}

		data.State = types.BlockState(state)
		devices[data.ID] = data
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

func (ds *sqliteDB) addBlockData(data types.Volume) error {
	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	err := ds.create("block_data", data.ID, data.TenantID, data.Size, string(data.State), data.CreateTime.Format(time.RFC3339Nano), data.Name, data.Description, data.Internal)

	return err
}

// For now we only support updating the state.
func (ds *sqliteDB) updateBlockData(data types.Volume) error {
	db := ds.getTableDB("block_data")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("UPDATE block_data SET state = ? WHERE id = ?", string(data.State), data.ID)

	return err
}

func (ds *sqliteDB) deleteBlockData(ID string) error {
	db := ds.getTableDB("block_data")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("DELETE FROM block_data WHERE id = ?", ID)

	return err
}

func (ds *sqliteDB) addStorageAttachment(a types.StorageAttachment) error {
	db := ds.getTableDB("attachments")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("INSERT INTO attachments (id, instance_id, block_id, ephemeral, boot) VALUES (?, ?, ?, ?, ?)", a.ID, a.InstanceID, a.BlockID, a.Ephemeral, a.Boot)

	return err
}

func (ds *sqliteDB) getAllStorageAttachments() (map[string]types.StorageAttachment, error) {
	attachments := make(map[string]types.StorageAttachment)

	db := ds.getTableDB("attachments")

	query := `SELECT	attachments.id,
				attachments.instance_id,
				attachments.block_id,
				attachments.ephemeral,
				attachments.boot
		  FROM	attachments `

	rows, err := db.Query(query)
	if err != nil {
		return attachments, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var a types.StorageAttachment

		err = rows.Scan(&a.ID, &a.InstanceID, &a.BlockID, &a.Ephemeral, &a.Boot)
		if err != nil {
			continue
		}
		attachments[a.ID] = a
	}

	if err = rows.Err(); err != nil {
		return attachments, err
	}

	return attachments, nil
}

func (ds *sqliteDB) deleteStorageAttachment(ID string) error {
	db := ds.getTableDB("attachments")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("DELETE FROM attachments WHERE id = ?", ID)

	return err
}

// this is here just for readability.
func (ds *sqliteDB) addPool(pool types.Pool) error {
	return ds.updatePool(pool)
}

// lock must be held by caller. Any rollbacks will need to be handled
// by caller.
func (ds *sqliteDB) updateSubnets(tx *sql.Tx, pool types.Pool) error {
	// get currently known subnets.
	subnets, err := ds.getPoolSubnets(pool.ID)
	if err != nil {
		// TBD: what about row not found?
		return err
	}

	// make a map of pool subnets by ID
	subMap := make(map[string]bool)
	for _, sub := range pool.Subnets {
		subMap[sub.ID] = true
	}

	// do we have any subnets that need deleting?
	for _, sub := range subnets {
		_, ok := subMap[sub.ID]
		if !ok {
			_, err = tx.Exec("DELETE FROM subnet_pool WHERE id = ?", sub.ID)
			if err != nil {
				return err
			}
		}
	}

	// any subnets that already exist in the table will be ignored,
	// new ones will be added.
	for _, subnet := range pool.Subnets {
		_, err = tx.Exec("INSERT OR IGNORE INTO subnet_pool (id, pool_id, cidr) VALUES (?, ?, ?)", subnet.ID, pool.ID, subnet.CIDR)
		if err != nil {
			return err
		}
	}

	return nil
}

// lock must be held by caller. Any rollbacks will need to be handled
// by caller.
func (ds *sqliteDB) updateAddresses(tx *sql.Tx, pool types.Pool) error {
	// get currently known individual addresses.
	addresses, err := ds.getPoolAddresses(pool.ID)
	if err != nil {
		// TBD: what about row not found?
		return err
	}

	// make a map of pool addresses by ID
	addrMap := make(map[string]bool)
	for _, addr := range pool.IPs {
		addrMap[addr.ID] = true
	}

	// do we have any individual IPs that need deleting?
	for _, addr := range addresses {
		_, ok := addrMap[addr.ID]
		if !ok {
			_, err = tx.Exec("DELETE FROM address_pool WHERE id = ?", addr.ID)
			if err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}

	// any addresses that already exist in the table will be ignored,
	// new ones will be added.
	for _, IP := range pool.IPs {
		_, err = tx.Exec("INSERT OR IGNORE INTO address_pool (id, pool_id, address) VALUES (?, ?, ?)", IP.ID, pool.ID, IP.Address)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return nil
}

// updatePool is used to update all pool related fields even if they
// are in different tables.
func (ds *sqliteDB) updatePool(pool types.Pool) error {
	db := ds.getTableDB("pools")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	pools := ds.getAllPools()

	// do the below as a single transaction.
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	err = ds.updateSubnets(tx, pool)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = ds.updateAddresses(tx, pool)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	// if this is a new pool, put it in, otherwise just update.
	_, ok := pools[pool.ID]
	if !ok {
		_, err = tx.Exec("INSERT INTO pools (id, name, free, total) VALUES (?, ?, ?, ?)", pool.ID, pool.Name, pool.Free, pool.TotalIPs)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	} else {
		// update free and total counts.
		_, err = tx.Exec("UPDATE pools SET free = ?, total = ? WHERE id = ?", pool.Free, pool.TotalIPs, pool.ID)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	err = tx.Commit()

	return err
}

func (ds *sqliteDB) getAllPools() map[string]types.Pool {
	pools := make(map[string]types.Pool)

	db := ds.getTableDB("pools")

	query := `SELECT	id,
				name,
				free,
				total
		  FROM	pools`

	rows, err := db.Query(query)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var pool types.Pool

		err = rows.Scan(&pool.ID, &pool.Name, &pool.Free, &pool.TotalIPs)
		if err != nil {
			continue
		}

		pool.Subnets, err = ds.getPoolSubnets(pool.ID)
		if err != nil {
			continue
		}

		pool.IPs, err = ds.getPoolAddresses(pool.ID)
		if err != nil {
			continue
		}

		pools[pool.ID] = pool
	}

	if err = rows.Err(); err != nil {
		return nil
	}

	return pools
}

func (ds *sqliteDB) deletePool(ID string) error {
	db := ds.getTableDB("pools")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// lock is held here and ok because the
	// get functions don't hold a lock.
	subnets, err := ds.getPoolSubnets(ID)
	if err != nil {
		return err
	}

	IPs, err := ds.getPoolAddresses(ID)
	if err != nil {
		return err
	}

	for _, subnet := range subnets {
		_, err = tx.Exec("DELETE FROM subnet_pool WHERE id = ?", subnet.ID)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	for _, addr := range IPs {
		_, err = tx.Exec("DELETE FROM address_pool WHERE id = ?", addr.ID)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	_, err = tx.Exec("DELETE FROM pools WHERE id = ?", ID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = tx.Commit()

	return err
}

func (ds *sqliteDB) getPoolSubnets(poolID string) ([]types.ExternalSubnet, error) {
	var subnets []types.ExternalSubnet

	db := ds.getTableDB("subnet_pool")

	query := `SELECT	id,
				cidr
		  FROM	subnet_pool
		  WHERE pool_id = ?`

	rows, err := db.Query(query, poolID)
	if err != nil {
		return subnets, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var subnet types.ExternalSubnet

		err = rows.Scan(&subnet.ID, &subnet.CIDR)
		if err != nil {
			continue
		}

		subnets = append(subnets, subnet)
	}

	if err = rows.Err(); err != nil {
		return subnets, err
	}

	return subnets, nil
}

func (ds *sqliteDB) getPoolAddresses(poolID string) ([]types.ExternalIP, error) {
	var IPs []types.ExternalIP

	db := ds.getTableDB("address_pool")

	query := `SELECT	id,
				address
		  FROM	address_pool
		  WHERE pool_id = ?`

	rows, err := db.Query(query, poolID)
	if err != nil {
		return IPs, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var IP types.ExternalIP

		err = rows.Scan(&IP.ID, &IP.Address)
		if err != nil {
			continue
		}

		IPs = append(IPs, IP)
	}

	if err = rows.Err(); err != nil {
		return IPs, err
	}

	return IPs, nil
}

func (ds *sqliteDB) addMappedIP(m types.MappedIP) error {
	db := ds.getTableDB("mapped_ips")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("INSERT INTO mapped_ips (id, pool_id, external_ip, instance_id) VALUES (?, ?, ?, ?)", m.ID, m.PoolID, m.ExternalIP, m.InstanceID)

	return err
}

func (ds *sqliteDB) deleteMappedIP(ID string) error {
	db := ds.getTableDB("mapped_ips")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec("DELETE FROM mapped_ips WHERE id = ?", ID)

	return err
}

func (ds *sqliteDB) getMappedIPs() map[string]types.MappedIP {
	IPs := make(map[string]types.MappedIP)

	db := ds.getTableDB("mapped_ips")

	query := `SELECT	mapped_ips.id,
				mapped_ips.pool_id,
				mapped_ips.external_ip,
				mapped_ips.instance_id,
				instances.ip,
				instances.tenant_id,
				pools.name
		  FROM	mapped_ips
		  JOIN instances
		  ON instances.id = mapped_ips.instance_id
		  JOIN pools
		  ON pools.id = mapped_ips.pool_id`

	rows, err := db.Query(query)
	if err != nil {
		fmt.Println(err)
		return IPs
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var IP types.MappedIP

		err = rows.Scan(&IP.ID, &IP.PoolID, &IP.ExternalIP, &IP.InstanceID, &IP.InternalIP, &IP.TenantID, &IP.PoolName)
		if err != nil {
			continue
		}

		IPs[IP.ExternalIP] = IP
	}

	if err = rows.Err(); err != nil {
		fmt.Println(err)
	}

	return IPs
}

func (ds *sqliteDB) updateQuotas(tenantID string, qds []types.QuotaDetails) error {
	db := ds.getTableDB("quotas")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "error starting transaction for quota update")
	}

	for i := range qds {
		_, err = tx.Exec("REPLACE INTO quotas (tenant_id, name, value) VALUES (?, ?, ?)", tenantID, qds[i].Name, qds[i].Value)
		if err != nil {
			_ = tx.Rollback()
			return errors.Wrap(err, "error executing query for quota update")
		}
	}

	err = tx.Commit()

	return errors.Wrap(err, "error committing transaction for quotas update")
}

func (ds *sqliteDB) getQuotas(tenantID string) ([]types.QuotaDetails, error) {
	query := `SELECT name, value from quotas WHERE tenant_id = ?`

	db := ds.getTableDB("quotas")

	rows, err := db.Query(query, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "error getting quotas from database")
	}
	defer func() { _ = rows.Close() }()

	results := []types.QuotaDetails{}
	for rows.Next() {
		var name string
		var value int

		err = rows.Scan(&name, &value)
		if err != nil {
			return nil, errors.Wrap(err, "error reading quota row from database")
		}

		q := types.QuotaDetails{Name: name, Value: value}
		results = append(results, q)
	}

	return results, nil
}

func (ds *sqliteDB) getImages() ([]types.Image, error) {
	images := []types.Image{}

	query := `SELECT id, state, tenant_id, name, createtime, size, visibility FROM images`

	db := ds.getTableDB("images")
	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	rows, err := db.Query(query)
	if err != nil {
		return images, errors.Wrap(err, "error getting images from database")
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		i := types.Image{}
		var state, visibility string

		err = rows.Scan(&i.ID, &state, &i.TenantID, &i.Name, &i.CreateTime, &i.Size, &visibility)
		if err != nil {
			return []types.Image{}, errors.Wrap(err, "error reading image row from database")
		}

		i.State = types.ImageState(state)
		i.Visibility = types.Visibility(visibility)

		images = append(images, i)
	}

	return images, nil
}

func (ds *sqliteDB) updateImage(i types.Image) error {
	query := `REPLACE INTO images (id, state, tenant_id, name, createtime, size, visibility) VALUES (?, ?, ?, ?, ?, ?, ?)`

	db := ds.getTableDB("images")
	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec(query, i.ID, i.State, i.TenantID, i.Name, i.CreateTime, i.Size, i.Visibility)

	return errors.Wrap(err, "Error updatiing image into database")
}

func (ds *sqliteDB) deleteImage(ID string) error {
	query := `DELETE FROM images WHERE id = ?`

	db := ds.getTableDB("images")
	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	_, err := db.Exec(query, ID)

	return errors.Wrap(err, "Error deleting image from database")
}
