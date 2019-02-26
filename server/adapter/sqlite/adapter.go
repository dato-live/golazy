// +build sqlite

package sqlite

type adapter struct {
	db      *sqlite3.SQLiteConn
	dbName  string
	version string
}

const (
	defaultDatabase = "golazy.db"
	dbVersion       = "100"
	adapterName     = "sqlite"
)

func (a *adapter) Open(conf config.Config) error {
	var err error
	return err

}

// Close closes the underlying database connection
func (a *adapter) Close() error {
	var err error
	if a.db != nil {
		err = a.db.Close()
		a.db = nil
		a.version = ""
	}
	return err
}

// IsOpen returns true if connection to database has been established. It does not check if
// connection is actually live.
func (a *adapter) IsOpen() bool {
	return a.db != nil
}

// Read current database version
func (a *adapter) getDbVersion() (string, error) {
	var vers t.KvMeta
	_, err := a.db.Where("key=?", "version").Get(vers)
	if err != nil {
		if isMissingDb(err) || err == sql.ErrNoRows {
			err = errors.New("Database not initialized")
		}
		return "", err
	}
	a.version = vers.Value

	return a.version, nil
}

// CheckDbVersion checks whether the actual DB version matches the expected version of this adapter.
func (a *adapter) CheckDbVersion() error {
	if a.version == "" {
		_, err := a.getDbVersion()
		if err != nil {
			return err
		}
	}

	if a.version != dbVersion {
		return errors.New("Invalid database version " + a.version +
			". Expected " + dbVersion)
	}

	return nil
}

// GetName returns string that adapter uses to register itself with store.
func (a *adapter) GetName() string {
	return adapterName
}

// CreateDb initializes the storage.
func (a *adapter) CreateDb(reset bool) error {
	var err error
	return err

}

func init() {
	store.RegisterAdapter(adapterName, &adapter{})
}
