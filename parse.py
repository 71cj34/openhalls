import xml.etree.ElementTree as ET
import json
import requests
from pathlib import Path
from datetime import datetime, timedelta
import time

def tt():
    return int(time.time()) // 60 % 1000

def ee():
    a = tt()
    return a % 3 + a % 39 + a % 42

def process_course_data(path, semester_code):
    with open(path, "r", encoding="utf-8") as f:
        xml_string = f.read()
    root = ET.fromstring(xml_string)
    course_elem = root.find('.//course')
    offering = root.find('.//offering')

    Path(f"./state/{semester_code}").mkdir(parents=True, exist_ok=True)

    course_info = {
        "code": course_elem.attrib['key'],
        "title": offering.attrib['title'],
        "desc": offering.attrib['desc']
    }

    schedule_map = {} # { day: { room: [ {start, end, course_code, section} ] } }

    for sel in root.findall('.//uselection'):
        time_map = {tb.attrib['id']: tb.attrib for tb in sel.findall('timeblock')}

        for block in sel.findall('.//selection/block'):
            day_ids = block.attrib['timeblockids'].split(',')
            for tid in day_ids:
                if tid in time_map:
                    t = time_map[tid]
                    room = block.attrib.get('location') or "TBD"

                    entry = {
                        "start": int(t['t1']),
                        "end": int(t['t2']),
                        "course": course_info['code'],
                        "display": block.attrib['disp']
                    }

                    # Nesting: Day -> Room -> List of courses
                    day = t['day']
                    schedule_map.setdefault(day, {}).setdefault(room, []).append(entry)

    return {"course": course_info, "schedule": schedule_map}

def get_course_data(semester_code: str, data_dir: str = "."):
    json_path = Path(data_dir) / f"{semester_code}.json"
    Path("./xml").mkdir(parents=True, exist_ok=True)

    with open(json_path, "r", encoding="utf-8") as f:
        courses = json.load(f)

    base_url = "https://mytimetable.mcmaster.ca/api/class-data"
    results = {}

    for course in courses:
        if course.get("title", "").startswith("("):
            print(f"Skipping course {course['code']} (title starts with '(')")
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
            "_": "",
        }
        session = requests.Session()

        headers = {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:153.0) Gecko/20100101 Firefox/153.0",
            "Accept": "application/xml, text/xml, */*; q=0.01",
            "Accept-Language": "en-US,en;q=0.9",
            "Accept-Encoding": "gzip, deflate, br, zstd",
            "Referer": "https://mytimetable.mcmaster.ca/criteria.jsp?access=0&lang=en&tip=2&page=results&scratch=0&advice=0&legend=1&term=3202630&sort=none&filters=liiiiiiiii&bbs=&ds=&cams=MCMSTiMCMST_MCMSTiMHK_MCMSTiOFF_MCMSTiCON_MCMSTiSNPOL&locs=any&isrts=any&ses=any&pl=&pac=1",
            "Sec-Fetch-Dest": "empty",
            "Sec-Fetch-Mode": "cors",
            "Sec-Fetch-Site": "same-origin",
            "Cookie": '',
        }


        time.sleep(0.25)
        params["_"] = int(datetime.now().timestamp() * 1000)
        print(f"Faking with {params}")
        response = session.get(base_url, params=params, headers=headers)

        xml_filename = f"{course['code'].replace(' ', '_')}.xml"
        xml_path = Path(".") / "xml" / semester_code / xml_filename
        with open(xml_path, "w", encoding="utf-8") as f:
            f.write(response.text)

def process_data_to_schedules(semester_code: str, data_dir: str = "."):
    state_folder = Path(data_dir) / "state" / semester_code
    schedules_folder = Path(data_dir) / "schedules" / semester_code
    schedules_folder.mkdir(parents=True, exist_ok=True)

    building_schedule = {}  # { building: { room_id: [ {start, end, course, display} ] } }

    for json_file in state_folder.glob("*.json"):
        with open(json_file, "r", encoding="utf-8") as f:
            data = json.load(f)

        schedule = data.get("schedule", {})
        for day, rooms in schedule.items():
            for room_key, times in rooms.items():
                if " - " in room_key:
                    building, room_id = room_key.split(" - ", 1)
                else:
                    building, room_id = room_key, "Unknown"

                building_schedule.setdefault(building, {}).setdefault(room_id, []).extend(times)

    output_path = schedules_folder / f"{semester_code}.json"
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(building_schedule, f, indent=2)

    return output_path



if __name__ == "__main__":
    sem = "3202630"
    # get_course_data(sem, "courses")

    # Process all XML files in the xml folder
    # xml_folder = Path(".") / "xml" / sem
    # processed_folder = Path(".") / "state"
    # processed_folder.mkdir(exist_ok=True)
    # for xml_file in xml_folder.glob("*.xml"):
    #     print(f"Processing {xml_file.name}...")
    #     result = process_course_data(xml_file, sem)
    #     output_path = processed_folder / sem / f"{xml_file.stem}.json"
    #     with open(output_path, "w", encoding="utf-8") as f:
    #         json.dump(result, f, indent=2)
    #     print(f"Saved to {output_path}")

    process_data_to_schedules(sem)