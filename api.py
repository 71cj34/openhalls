import argparse
import datetime
import json
import re
import time
from pathlib import Path
from typing import Optional
import xml.etree.ElementTree as ET
import requests


def get_courses(sems: list[int], out_dir: str = "courses", delay: float = 0.25) -> list[Path]:
    headers = {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        "Accept": "text/xml",
    }
    out = Path(out_dir)
    out.mkdir(parents=True, exist_ok=True)

    saved = []
    for s in sems:
        print(f"--- Processing semester: {s} ---")
        courses = []
        n = 0
        cval = "20"

        params = {
            "term": s,
            "cams": "MCMSTiMCMST_MCMSTiMHK_MCMSTiOFF_MCMSTiCON_MCMSTiSNPOL",
            "course_add": " ",
            "page_num": n,
            "sco": 1,
            "sio": 1,
            "already": "",
        }

        while cval == "20":
            params["page_num"] = n
            params["_"] = int(datetime.datetime.now().timestamp() * 1000)

            time.sleep(delay)  # Be polite
            try:
                response = requests.get("https://mytimetable.mcmaster.ca/api/courses/suggestions", params=params, headers=headers, timeout=30)
            except requests.RequestException as exc:
                print(f"Failed to fetch page {n} for {s}: {exc}")
                break

            if response.status_code != 200:
                print(f"Failed to fetch page {n} for {s}: HTTP {response.status_code}")
                break

            try:
                root = ET.fromstring(response.text)
            except ET.ParseError as exc:
                print(f"Failed to parse page {n} for {s}: {exc}")
                break
            cval = root.text.strip() if root.text else ""

            for item in root.findall(".//rs"):
                course_code = item.text.strip() if item.text else ""
                if course_code.startswith("_") or not course_code:
                    continue

                raw_info = item.get("info", "")
                clean_info = re.sub(r"<[^>]+>", " ", raw_info).strip()
                if clean_info.endswith("-"):
                    clean_info = clean_info[:-1].strip()

                courses.append({"code": course_code, "title": clean_info})

            print(f"Fetched page {n} for {s}...")
            n += 1

        filename = out / f"{s}.json"
        with open(filename, "w", encoding="utf-8") as f:
            json.dump(courses, f, indent=4)
        print(f"Saved {len(courses)} courses to {filename}")
        saved.append(filename)
    return saved


def autosem(delay: float = 0.5) -> list[int]:
    y = datetime.date.today().year
    valid = []
    years = [y - 1, y, y + 1]
    post = [10, 20, 30]
    strings = [f"3{yr}{inc}" for yr in years for inc in post]
    for s in strings:
        time.sleep(delay)
        print(f"Trying {s}")
        try:
            dryfire = requests.get(
                f"https://mytimetable.mcmaster.ca/api/courses/suggestions?term={s}&cams=MCMSTiMCMST_MCMSTiMHK_MCMSTiOFF_MCMSTiCON_MCMSTiSNPOL&course_add=%20&page_num=0&sco=1&sio=1&already=&_={int(datetime.datetime.now().timestamp() * 1000)}",
                timeout=30,
            )
            root = ET.fromstring(dryfire.text)

            rs_elements = root.findall('.//rs')

            is_error = all('(' in rs.get('info', '') for rs in rs_elements)

            if not is_error and len(rs_elements) > 0:
                print(f"FOUND VALID: {s}")
                valid.append(int(s))

        except (ET.ParseError, requests.RequestException):
            continue

    return valid


def main(argv: Optional[list[str]] = None):
    p = argparse.ArgumentParser(
        description="Step 1: download the course catalogue (no login required)."
    )
    p.add_argument("--sem", nargs="*", default=[],
                   help="Semester code(s), e.g. --sem 3202630. If omitted with --auto, valid terms are probed automatically.")
    p.add_argument("--auto", action="store_true",
                   help="Auto-detect valid semester codes and fetch all of them.")
    p.add_argument("--out-dir", default="courses", help="Directory for <semester>.json files (default: courses).")
    p.add_argument("--delay", type=float, default=0.25, help="Seconds between requests (default: 0.25).")
    args = p.parse_args(argv)

    sems = [int(s) for s in args.sem]
    if args.auto or not sems:
        detected = autosem()
        if not detected:
            print("No valid semesters detected. Pass --sem explicitly, e.g. --sem 3202630")
            return
        sems = detected if args.auto or not sems else sems
        if args.auto:
            print(f"Using auto-detected semesters: {sems}")
    get_courses(sems, out_dir=args.out_dir, delay=args.delay)


if __name__ == "__main__":
    main()
