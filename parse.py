import argparse
import json
import os
import re
import time
import xml.etree.ElementTree as ET
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Tuple

import requests


BASE_URL = "https://mytimetable.mcmaster.ca/api/class-data"
COOKIE_ENV_VAR = "MYTIMETABLE_COOKIE"


def tt():
    return int(time.time()) // 60 % 1000


def ee():
    a = tt()
    return a % 3 + a % 39 + a % 42


def _cookie_from_dotenv(dotenv_path: str = ".env") -> str:
    p = Path(dotenv_path)
    if not p.is_file():
        return ""
    for line in p.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, _, value = line.partition("=")
        if key.strip() == COOKIE_ENV_VAR:
            return value.strip().strip('"').strip("'")
    return ""


def resolve_cookie(cli_cookie: str = "", cookie_file: str = "") -> str:
    if cli_cookie:
        return cli_cookie.strip()
    if cookie_file:
        cookie = Path(cookie_file).read_text(encoding="utf-8").strip()
        if cookie:
            return cookie
    env_cookie = os.environ.get(COOKIE_ENV_VAR, "").strip()
    if env_cookie:
        return env_cookie
    return _cookie_from_dotenv()


def _cookie_help() -> str:
    return (
        "A browser cookie is required for the class-data endpoint.\n"
        "1. Log in at https://mytimetable.mcmaster.ca in your browser.\n"
        "2. Open DevTools > Network, run any course search.\n"
        "3. Click a 'class-data' request > Request Headers > copy the "
        "'Cookie:' value.\n"
        f"4. Pass it via --cookie \"...\", --cookie-file cookie.txt, "
        f"the {COOKIE_ENV_VAR} env var, or a .env file with "
        f"{COOKIE_ENV_VAR}=.... Cookies expire, so repeat when requests fail."
    )


def process_course_data(path, semester_code):
    with open(path, "r", encoding="utf-8") as f:
        xml_string = f.read()
    try:
        root = ET.fromstring(xml_string)
    except ET.ParseError as exc:
        raise ValueError(f"{path}: not valid XML ({exc}). Cookie may be expired.") from exc
    course_elem = root.find('.//course')
    offering = root.find('.//offering')
    if course_elem is None or offering is None:
        raise ValueError(
            f"{path}: no <course>/<offering> found. Response was likely an "
            "error page (bad/expired cookie) rather than class data."
        )

    course_info = {
        "code": course_elem.attrib['key'],
        "title": offering.attrib.get('title', ''),
        "desc": offering.attrib.get('desc', '')
    }

    schedule_map = {}  # { day: { room: [ {start, end, course_code, section} ] } }
    seen = set()  # (day, room, start, end, course, display)

    for sel in root.findall('.//uselection'):
        time_map = {tb.attrib['id']: tb.attrib for tb in sel.findall('timeblock')}

        for block in sel.findall('.//selection/block'):
            raw_tids = (block.attrib.get('timeblockids') or '').strip()
            if not raw_tids:
                continue
            try:
                loos = json.loads(block.attrib.get('loos') or '{}')
            except (json.JSONDecodeError, TypeError):
                loos = {}
            if not isinstance(loos, dict):
                loos = {}
            fallback_room = block.attrib.get('location') or "TBD"

            for tid in raw_tids.split(','):
                tid = tid.strip()
                if not tid or tid not in time_map:
                    continue
                t = time_map[tid]
                # locs maps tid -> room for split-location blocks (e.g. Tue in JHE, Thu in ETB)
                room = (loos.get(tid) or fallback_room or "TBD").strip() or "TBD"

                try:
                    start, end = int(t['t1']), int(t['t2'])
                except (KeyError, TypeError, ValueError):
                    continue
                day = t.get('day')
                if day is None:
                    continue
                display = block.attrib.get('disp') or ''

                key = (day, room, start, end, course_info['code'], display)
                if key in seen:
                    continue
                seen.add(key)

                entry = {
                    "start": start,
                    "end": end,
                    "course": course_info['code'],
                    "display": display
                }

                # Day -> Room -> List of courses
                schedule_map.setdefault(day, {}).setdefault(room, []).append(entry)

    return {"course": course_info, "schedule": schedule_map}


def get_course_data(
    semester_code: str,
    courses_dir: str = "courses",
    xml_dir: str = "xml",
    course_file: Optional[str] = None,
    cookie: str = "",
    limit: Optional[int] = None,
    skip_existing: bool = True,
    delay: float = 0.25,
) -> Tuple[List[Path], List[str]]:
    if not cookie:
        raise SystemExit(
            "Missing browser cookie, cannot call class-data.\n" + _cookie_help()
        )
    json_path = Path(course_file) if course_file else Path(courses_dir) / f"{semester_code}.json"
    if not json_path.is_file():
        raise SystemExit(
            f"Course list not found: {json_path}\n"
            f"Run step 1 first: python api.py --sem {semester_code}"
        )
    with open(json_path, "r", encoding="utf-8") as f:
        courses = json.load(f)
    if limit is not None:
        courses = courses[:limit]

    xml_folder = Path(xml_dir) / str(semester_code)
    xml_folder.mkdir(parents=True, exist_ok=True)

    session = requests.Session()
    session.headers.update({
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:153.0) Gecko/20100101 Firefox/153.0",
        "Accept": "application/xml, text/xml, */*; q=0.01",
        "Accept-Language": "en-US,en;q=0.9",
        "Accept-Encoding": "gzip, deflate, br, zstd",
        "Referer": "https://mytimetable.mcmaster.ca/criteria.jsp?access=0&lang=en&tip=2&page=results&scratch=0&advice=0&legend=1&term=3202630&sort=none&filters=liiiiiiiii&bbs=&ds=&cams=MCMSTiMCMST_MCMSTiMHK_MCMSTiOFF_MCMSTiCON_MCMSTiSNPOL&locs=any&isrts=any&ses=any&pl=&pac=1",
        "Sec-Fetch-Dest": "empty",
        "Sec-Fetch-Mode": "cors",
        "Sec-Fetch-Site": "same-origin",
        "Cookie": cookie,
    })

    saved, failed = [], []
    total = len(courses)
    for i, course in enumerate(courses, 1):
        if course.get("title", "").startswith("("):
            print(f"[{i}/{total}] Skipping {course['code']} (no sections this term)")
            continue

        xml_filename = f"{course['code'].replace(' ', '_')}.xml"
        xml_path = xml_folder / xml_filename
        if skip_existing and xml_path.is_file() and xml_path.stat().st_size > 0:
            print(f"[{i}/{total}] Exists, skipping {course['code']}")
            saved.append(xml_path)
            continue

        course_code = course["code"].replace(" ", "-")
        params = {
            "term": semester_code,
            "course_0_0": course_code,
            "va_0_0": "e2ed",
            "rq_0_0": "",
            "t": tt(),
            "e": ee(),
            "nouser": "1",
            "_": int(datetime.now().timestamp() * 1000),
        }

        time.sleep(delay)
        try:
            response = session.get(BASE_URL, params=params, timeout=30)
            response.raise_for_status()
        except requests.RequestException as exc:
            print(f"[{i}/{total}] FAILED {course['code']}: {exc}")
            failed.append(course["code"])
            continue

        if "<course " not in response.text:
            print(f"[{i}/{total}] WARNING {course['code']}: no class data returned "
                  "(cookie may be expired). Saved for inspection.")
            failed.append(course["code"])
        else:
            print(f"[{i}/{total}] Fetched {course['code']}")
        xml_path.write_text(response.text, encoding="utf-8")
        saved.append(xml_path)

    if failed:
        print(f"\n{len(failed)} course(s) need attention: {', '.join(failed[:10])}"
              f"{'...' if len(failed) > 10 else ''}")
        print("If many failed at once, refresh your browser cookie and re-run "
              "(existing files are kept unless --force).")
    return saved, failed


def process_all_xml(semester_code: str, xml_dir: str = "xml",
                    state_dir: str = "state") -> Tuple[List[Path], List[str]]:
    """Convert every xml/<sem>/*.xml to state/<sem>/*.json."""
    xml_folder = Path(xml_dir) / str(semester_code)
    state_folder = Path(state_dir) / str(semester_code)
    state_folder.mkdir(parents=True, exist_ok=True)

    xml_files = sorted(xml_folder.glob("*.xml"))
    if not xml_files:
        raise SystemExit(f"No XML files in {xml_folder}. Run with fetching enabled first.")

    saved, failed = [], []
    for xml_file in xml_files:
        try:
            result = process_course_data(xml_file, semester_code)
        except (ValueError, OSError) as exc:
            print(f"Skipping {xml_file.name}: {exc}")
            failed.append(xml_file.name)
            continue
        output_path = state_folder / f"{xml_file.stem}.json"
        with open(output_path, "w", encoding="utf-8") as f:
            json.dump(result, f, indent=2)
        saved.append(output_path)
    print(f"Processed {len(saved)}/{len(xml_files)} XML files -> {state_folder}")
    return saved, failed


def process_data_to_schedules(semester_code: str, state_dir: str = "state",
                              schedules_dir: str = "schedules"):
    state_folder = Path(state_dir) / str(semester_code)
    schedules_folder = Path(schedules_dir) / str(semester_code)
    schedules_folder.mkdir(parents=True, exist_ok=True)

    if not state_folder.is_dir():
        raise SystemExit(f"No parsed data in {state_folder}. Run the process step first.")

    building_schedule: Dict = {}  # { building: { room_id: [ {day, start, end, course, display} ] } }
    seen = set()

    files = list(state_folder.glob("*.json"))
    for json_file in files:
        with open(json_file, "r", encoding="utf-8") as f:
            data = json.load(f)

        schedule = data.get("schedule", {})
        for day, rooms in schedule.items():
            for room_key, times in rooms.items():
                room_entries = room_key.split("; ")
                for entry in room_entries:
                    if " - " in entry:
                        building, room_id = entry.split(" - ", 1)
                    else:
                        building, room_id = entry, "Unknown"

                    for time_entry in times:
                        key = (building, room_id, day, time_entry.get("start"),
                               time_entry.get("end"), time_entry.get("course"),
                               time_entry.get("display"))
                        if key in seen:
                            continue
                        seen.add(key)
                        enriched = {**time_entry, "day": day}
                        building_schedule \
                            .setdefault(building, {}) \
                            .setdefault(room_id, []) \
                            .append(enriched)

    output_path = schedules_folder / f"{semester_code}.json"
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(building_schedule, f, indent=2)

    print(f"Built {output_path} from {len(files)} course files "
          f"({len(building_schedule)} buildings)")
    return output_path


def main(argv: Optional[List[str]] = None):
    p = argparse.ArgumentParser(
        description="Fetch class XML and build room schedules. "
                    "Default runs fetch -> process -> build.",
        epilog="Cookie: pass --cookie, --cookie-file, "
               f"{COOKIE_ENV_VAR} env var, or .env. Only needed for fetching.",
    )
    p.add_argument("--semester", required=True, help="Semester code, e.g. 3202630.")
    p.add_argument("--courses-dir", default="courses")
    p.add_argument("--course-file", default=None,
                   help="Override course list (default: <courses-dir>/<semester>.json). "
                        "Use sample_data.json for a quick test.")
    p.add_argument("--xml-dir", default="xml")
    p.add_argument("--state-dir", default="state")
    p.add_argument("--schedules-dir", default="schedules")
    p.add_argument("--cookie", default="",
                   help="Browser Cookie header value for class-data.")
    p.add_argument("--cookie-file", default="",
                   help="File containing the Cookie header value.")
    p.add_argument("--limit", type=int, default=None,
                   help="Fetch only the first N courses (smoke test).")
    p.add_argument("--force", action="store_true",
                   help="Re-fetch XML files that already exist.")
    p.add_argument("--delay", type=float, default=0.25,
                   help="Seconds between requests (default: 0.25).")
    p.add_argument("--no-fetch", action="store_true", help="Skip downloading XML.")
    p.add_argument("--no-process", action="store_true", help="Skip XML -> state JSON.")
    p.add_argument("--no-build", action="store_true", help="Skip state -> schedule JSON.")
    args = p.parse_args(argv)

    cookie = resolve_cookie(args.cookie, args.cookie_file)
    if not args.no_fetch and not cookie:
        p.error("fetching needs a cookie.\n" + _cookie_help())

    if not args.no_fetch:
        get_course_data(
            args.semester,
            courses_dir=args.courses_dir,
            xml_dir=args.xml_dir,
            course_file=args.course_file,
            cookie=cookie,
            limit=args.limit,
            skip_existing=not args.force,
            delay=args.delay,
        )
    if not args.no_process:
        process_all_xml(args.semester, xml_dir=args.xml_dir, state_dir=args.state_dir)
    if not args.no_build:
        process_data_to_schedules(args.semester, state_dir=args.state_dir,
                                  schedules_dir=args.schedules_dir)


if __name__ == "__main__":
    main()
