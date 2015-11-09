// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2014-2015 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package asserts

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"golang.org/x/crypto/openpgp/packet"
	. "gopkg.in/check.v1"

	"github.com/ubuntu-core/snappy/helpers"
)

func Test(t *testing.T) { TestingT(t) }

type openSuite struct{}

var _ = Suite(&openSuite{})

func (opens *openSuite) TestOpenDatabaseOK(c *C) {
	rootDir := filepath.Join(c.MkDir(), "asserts-db")
	cfg := &DatabaseConfig{Path: rootDir}
	db, err := OpenDatabase(cfg)
	c.Assert(err, IsNil)
	c.Assert(db, NotNil)
	c.Check(db.root, Equals, rootDir)
	info, err := os.Stat(rootDir)
	c.Assert(err, IsNil)
	c.Assert(info.IsDir(), Equals, true)
	c.Check(info.Mode().Perm(), Equals, os.FileMode(0775))
}

func (opens *openSuite) TestOpenDatabaseWorldWriteableFail(c *C) {
	rootDir := filepath.Join(c.MkDir(), "asserts-db")
	oldUmask := syscall.Umask(0)
	os.MkdirAll(rootDir, 0777)
	syscall.Umask(oldUmask)
	cfg := &DatabaseConfig{Path: rootDir}
	db, err := OpenDatabase(cfg)
	c.Assert(err, Equals, ErrDbRootWorldReadable)
	c.Check(db, IsNil)
}

type databaseSuite struct {
	rootDir string
	db      *Database
}

var _ = Suite(&databaseSuite{})

func (dbs *databaseSuite) SetUpTest(c *C) {
	dbs.rootDir = filepath.Join(c.MkDir(), "asserts-db")
	cfg := &DatabaseConfig{Path: dbs.rootDir}
	db, err := OpenDatabase(cfg)
	c.Assert(err, IsNil)
	dbs.db = db
}

func (dbs *databaseSuite) TestAtomicWriteEntrySecret(c *C) {
	err := dbs.db.atomicWriteEntry([]byte("foobar"), true, "a", "b", "foo")
	c.Assert(err, IsNil)
	fooPath := filepath.Join(dbs.rootDir, "a", "b", "foo")
	info, err := os.Stat(fooPath)
	c.Assert(err, IsNil)
	c.Check(info.Mode().Perm(), Equals, os.FileMode(0600))
	c.Check(info.Size(), Equals, int64(6))
}

func (dbs *databaseSuite) TestImportKey(c *C) {
	privk, err := generatePrivateKey()
	c.Assert(err, IsNil)
	expectedFingerprint := hex.EncodeToString(privk.PublicKey.Fingerprint[:])

	fingerp, err := dbs.db.ImportKey("account0", privk)
	c.Assert(err, IsNil)
	c.Check(fingerp, Equals, expectedFingerprint)

	keyPath := filepath.Join(dbs.rootDir, privateKeysRoot, "account0", fingerp)
	info, err := os.Stat(keyPath)
	c.Assert(err, IsNil)
	c.Check(info.Mode().Perm(), Equals, os.FileMode(0600)) // secret
	// too white box? ok at least until we have more functionality
	fpriv, err := os.Open(keyPath)
	c.Assert(err, IsNil)
	pk, err := packet.Read(fpriv)
	c.Assert(err, IsNil)
	privKeyFromDisk, ok := pk.(*packet.PrivateKey)
	c.Assert(ok, Equals, true)
	c.Check(hex.EncodeToString(privKeyFromDisk.PublicKey.Fingerprint[:]), Equals, expectedFingerprint)
}

func (dbs *databaseSuite) TestGenerateKey(c *C) {
	fingerp, err := dbs.db.GenerateKey("account0")
	c.Assert(err, IsNil)
	c.Check(fingerp, NotNil)
	keyPath := filepath.Join(dbs.rootDir, privateKeysRoot, "account0", fingerp)
	c.Check(helpers.FileExists(keyPath), Equals, true)
}
