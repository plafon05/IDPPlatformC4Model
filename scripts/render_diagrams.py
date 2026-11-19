"""Render changed PlantUML sources and track the images owned by each source."""

import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parents[1]
OUTPUT = ROOT / "diagrams"
MANIFEST = OUTPUT / ".manifest.json"


def tracked_files():
    return [Path(os.fsdecode(name)) for name in subprocess.check_output(
        ["git", "ls-files", "-z"], cwd=ROOT
    ).split(b"\0") if name]


def fingerprint(path, seen=None):
    """Include local !include dependencies in the source fingerprint."""
    seen = set() if seen is None else seen
    path = path.resolve()
    if path in seen:
        return b""
    seen.add(path)
    data = path.read_bytes()
    digest = hashlib.sha256(data)
    for line in data.decode("utf-8-sig").splitlines():
        parts = line.strip().split(maxsplit=1)
        if len(parts) != 2 or parts[0] not in (
            "!include", "!include_once", "!include_many"
        ):
            continue
        target = parts[1].strip().strip('"')
        dependency = path.parent / target
        if dependency.is_file():
            digest.update(fingerprint(dependency, seen))
    return digest.digest()


def image_path(relative):
    path = (OUTPUT / relative).resolve()
    if not path.is_relative_to(OUTPUT.resolve()) or path.suffix not in (".png", ".svg"):
        raise ValueError(f"Invalid generated image path: {relative}")
    return path


def main():
    OUTPUT.mkdir(exist_ok=True)
    previous = json.loads(MANIFEST.read_text()) if MANIFEST.exists() else {}
    current = {}
    jar = os.environ["PLANTUML_JAR"]
    for source in sorted(p for p in tracked_files() if p.suffix == ".puml"):
        key = source.as_posix()
        digest = fingerprint(ROOT / source).hex()
        old = previous.get(key, {})
        if old.get("hash") == digest and old.get("images") and all(
            image_path(name).is_file() for name in old["images"]
        ):
            current[key] = old
            print(f"Unchanged: {source}", flush=True)
            continue
        print(f"Rendering: {source}", flush=True)
        with tempfile.TemporaryDirectory() as temporary:
            for extension in ("svg", "png"):
                subprocess.run([
                    "java", "-Djava.awt.headless=true", "-jar", jar,
                    "-charset", "UTF-8", "-failfast2", f"-t{extension}",
                    "-o", temporary, str(ROOT / source),
                ], check=True, cwd=ROOT)
            images = []
            for generated in sorted(Path(temporary).rglob("*")):
                if generated.suffix not in (".svg", ".png"):
                    continue
                relative = (source.parent / generated.relative_to(temporary)).as_posix()
                destination = image_path(relative)
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(generated, destination)
                images.append(relative)
            if not images:
                raise RuntimeError(f"No images generated for {source}")
            current[key] = {"hash": digest, "images": images}

    # Remove only images no longer produced by any source (including deleted sources).
    retained = {name for entry in current.values() for name in entry["images"]}
    obsolete = {name for entry in previous.values() for name in entry["images"]} - retained
    for name in sorted(obsolete):
        image_path(name).unlink(missing_ok=True)
        print(f"Removed: {name}", flush=True)
    MANIFEST.write_text(json.dumps(current, ensure_ascii=False, indent=2) + "\n")


if __name__ == "__main__":
    main()
