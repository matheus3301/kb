package main

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func readOperationalFixture(t *testing.T, base, name string) []byte {
	t.Helper()
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	data, readErr := root.ReadFile(name)
	if err := errors.Join(readErr, root.Close()); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestOperationalBackupCLIAndMCPParity(t *testing.T) {
	root := realFixture(t)
	exact := []byte("---\r\ntype: Reference # preserve comment\r\nprecision: 123456789.0000000001\r\nunknown: {nested: true}\r\n---\r\n# Exact source\r\nBody.\r\n")
	if err := os.WriteFile(filepath.Join(root, "exact.md"), exact, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	asset := []byte{0, 255, 1, 2, 3}
	if err := os.WriteFile(filepath.Join(root, "assets/source.bin"), asset, 0600); err != nil {
		t.Fatal(err)
	}
	create := `{"path":"03_Resources/replay.md","type":"Concept","body":"# Retry evidence\nA synthetic durable request.\n","actor":"process:backup-test","summary":"Record durable retry evidence","idempotency_key":"operational-replay"}`
	if code, e := invoke(t, []string{"--root", root, "create", "--input", "-"}, create); code != 0 {
		t.Fatalf("create: %#v", e)
	}
	cs, closeSession := realSession(t, root, false, nil)
	defer closeSession()
	code, cli := invoke(t, []string{"--root", root, "backup", ".kb/backups/cli.tar"}, "")
	if code != 0 {
		t.Fatalf("backup: %#v", cli)
	}
	protocol := protocolCall(t, cs, "backup", `{"archive":".kb/backups/mcp.tar"}`)
	if protocol["ok"] != true {
		t.Fatalf("MCP backup: %#v", protocol)
	}
	cliData := cli["data"].(map[string]any)
	mcpData := protocol["data"].(map[string]any)
	delete(cliData, "archive")
	delete(mcpData, "archive")
	if !reflect.DeepEqual(cliData, mcpData) {
		t.Fatalf("backup parity differs: %#v %#v", cliData, mcpData)
	}
	first := readOperationalFixture(t, root, ".kb/backups/cli.tar")
	second := readOperationalFixture(t, root, ".kb/backups/mcp.tar")
	if !bytes.Equal(first, second) {
		t.Fatal("CLI and MCP archives differ for the same source")
	}
	code, cli = invoke(t, []string{"--root", root, "restore", ".kb/backups/cli.tar", ".kb/restores/cli"}, "")
	if code != 0 {
		t.Fatalf("CLI restore: %#v", cli)
	}
	protocol = protocolCall(t, cs, "restore", `{"archive":".kb/backups/mcp.tar","destination":".kb/restores/mcp"}`)
	if protocol["ok"] != true {
		t.Fatalf("MCP restore: %#v", protocol)
	}
	cliData = cli["data"].(map[string]any)
	mcpData = protocol["data"].(map[string]any)
	for _, data := range []map[string]any{cliData, mcpData} {
		delete(data, "archive")
		delete(data, "destination")
	}
	if !reflect.DeepEqual(cliData, mcpData) {
		t.Fatalf("restore parity differs: %#v %#v", cliData, mcpData)
	}
	archive := tar.NewReader(bytes.NewReader(first))
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Name == "kb-backup-manifest.json" {
			continue
		}
		original, err := io.ReadAll(archive)
		if err != nil {
			t.Fatal(err)
		}
		for _, base := range []string{root, filepath.Join(root, ".kb/restores/cli"), filepath.Join(root, ".kb/restores/mcp")} {
			got := readOperationalFixture(t, base, header.Name)
			if !bytes.Equal(original, got) {
				t.Fatalf("%s: byte preservation failed", header.Name)
			}
		}
	}
	for _, name := range []string{"cli", "mcp"} {
		restoredRoot := filepath.Join(root, ".kb/restores", name)
		code, e := invoke(t, []string{"--root", restoredRoot, "create", "--input", "-"}, create)
		if code != 0 || e["data"].(map[string]any)["replayed"] != true {
			t.Fatalf("restored receipt replay failed: %#v", e)
		}
		if code, e := invoke(t, []string{"--root", restoredRoot, "read", "exact.md"}, ""); code != 0 || e["data"].(map[string]any)["markdown"] != string(exact) {
			t.Fatalf("exact read failed: %#v", e)
		}
		if code, e := invoke(t, []string{"--root", restoredRoot, "index", "rebuild"}, ""); code != 0 {
			t.Fatalf("cache rebuild failed: %#v", e)
		}
		if code, e := invoke(t, []string{"--root", restoredRoot, "search", "Retry"}, ""); code != 0 || e["data"].(map[string]any)["total"] != float64(1) {
			t.Fatalf("restored search failed: %#v", e)
		}
		request := `{"path":"after.md","type":"Concept","actor":"test","summary":"Continue after restoration","idempotency_key":"after"}`
		if code, e := invoke(t, []string{"--root", restoredRoot, "create", "--input", "-"}, request); code != 0 {
			t.Fatalf("normal write failed: %#v", e)
		}
	}
	for _, action := range []string{"backup", "restore"} {
		raw := `{"archive":".kb/backups/cli.tar"}`
		if action == "restore" {
			raw = `{"archive":".kb/backups/cli.tar","destination":".kb/restores/cli"}`
		}
		_, cli := invoke(t, []string{"--root", root, action, "--input", "-"}, raw)
		protocol := protocolCall(t, cs, action, raw)
		if !reflect.DeepEqual(cli, protocol) || cli["error"].(map[string]any)["code"] != "DESTINATION_EXISTS" {
			t.Fatalf("output conflict parity: %#v %#v", cli, protocol)
		}
	}
	readonly, closeReadonly := realSession(t, root, true, nil)
	defer closeReadonly()
	tools, err := readonly.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "kb_backup" || tool.Name == "kb_restore" {
			t.Fatal("read-only advertised operational writes")
		}
	}
	for _, action := range []string{"backup", "restore"} {
		_, cli := invoke(t, []string{"--root", root, "--read-only", action, "--input", "-"}, `{}`)
		protocol := protocolCall(t, readonly, action, `{}`)
		if !reflect.DeepEqual(cli, protocol) || cli["error"].(map[string]any)["code"] != "READ_ONLY" {
			t.Fatalf("readonly parity: %#v %#v", cli, protocol)
		}
	}
	// A model cannot turn either operator tool into arbitrary filesystem access.
	for _, request := range []map[string]any{{"archive": "../outside.tar"}, {"archive": "/absolute.tar"}, {"archive": ".kb/backups/cli.tar", "destination": "."}, {"archive": ".kb/backups/cli.tar", "destination": ".kb/restores/../live"}} {
		raw, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		action := "backup"
		if _, ok := request["destination"]; ok {
			action = "restore"
		}
		_, cli := invoke(t, []string{"--root", root, action, "--input", "-"}, string(raw))
		protocol := protocolCall(t, cs, action, string(raw))
		if cli["ok"] != false || !reflect.DeepEqual(cli, protocol) {
			t.Fatalf("confinement parity: %#v %#v", cli, protocol)
		}
	}
	// Existing source bytes remain exact after both restore roots accept new writes.
	got := readOperationalFixture(t, root, "exact.md")
	if !bytes.Equal(got, exact) {
		t.Fatal("source note changed")
	}
	got = readOperationalFixture(t, root, "assets/source.bin")
	if !bytes.Equal(got, asset) {
		t.Fatal("source asset changed")
	}
}
