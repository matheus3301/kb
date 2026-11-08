"""Exercise the public OKF library from an unrelated Go module."""

import json
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
PACKAGE = "github.com/matheus3301/kb/pkg/okf"


def dependency_findings(packages):
    allowed = {PACKAGE, "go.yaml.in/yaml/v3"}
    return [f"OKF runtime dependency is outside its format boundary: {item['ImportPath']}"
            for item in packages if not item.get("Standard") and item["ImportPath"] not in allowed]


def decode_packages(stream):
    decoder = json.JSONDecoder()
    packages = []
    while stream.strip():
        item, end = decoder.raw_decode(stream.lstrip())
        packages.append(item)
        stream = stream.lstrip()[end:]
    return packages


CONSUMER = r'''package main

import (
    "encoding/json"
    "fmt"
    "github.com/matheus3301/kb/pkg/okf"
)

func main() {
    raw := []byte("---\ntype: 'Reference' # Keep\nnumber: 1234567890.123456789\n---\nBody\n")
    document, err := okf.Parse(raw)
    if err != nil { panic(err) }
    if document.Metadata()["number"] != json.Number("1234567890.123456789") { panic("rounded number") }
    if len(document.Validate()) != 0 { panic("invalid concept") }
    body := "Revised\n"
    out, err := document.Edit(okf.Edit{Set: map[string]any{"new": nil}, Body: &body})
    if err != nil { panic(err) }
    fresh, err := okf.Parse(out)
    if err != nil { panic(err) }
    if value, exists := fresh.Metadata()["new"]; !exists || value != nil { panic("missing null") }
    if fresh.Body() != body || fresh.Metadata()["generated"] != nil { panic("unexpected policy") }
    if len(okf.ValidateReserved(okf.Log, []byte("## 2026-01-02\n## 2026-01-01\n"))) != 0 { panic("invalid log") }
    fmt.Println("external OKF consumer passed")
}
'''


def main():
    env = dict(os.environ, GOWORK="off", GOTOOLCHAIN="local")
    env.pop("KB_ROOT", None)
    stream = subprocess.check_output(["go", "list", "-deps", "-json", "./pkg/okf"], cwd=ROOT, env=env, text=True)
    findings = dependency_findings(decode_packages(stream))
    if findings:
        raise SystemExit("\n".join(findings))
    version = subprocess.check_output(["go", "list", "-m", "-f", "{{.GoVersion}}"], cwd=ROOT, env=env, text=True).strip()
    with tempfile.TemporaryDirectory(prefix="okf-consumer-") as temporary:
        directory = Path(temporary)
        # A replacement binds this smoke run to checked-out source, not a tag.
        (directory / "go.mod").write_text(
            f"module example.invalid/okf-consumer\n\ngo {version}\n\n"
            f"require github.com/matheus3301/kb v0.0.0\n\n"
            f"replace github.com/matheus3301/kb => {json.dumps(str(ROOT))}\n")
        (directory / "main.go").write_text(CONSUMER)
        subprocess.run(["go", "run", "-mod=mod", "."], cwd=directory, env=env, check=True)
    print("OKF dependency boundary passed: standard library + go.yaml.in/yaml/v3")


if __name__ == "__main__":
    main()
