import datetime
import re
import time
from typing import Optional
import xml.etree.ElementTree as ET
import requests
import json
from pathlib import Path
# from parse import process_course_data

def get_courses(sems: list[int]):
    url = "https://mytimetable.mcmaster.ca/api/courses/suggestions"
    headers = {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        "Accept": "text/xml",
    }

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

            time.sleep(0.25) # Be polite
            response = requests.get(url, params=params, headers=headers)

            if response.status_code != 200:
                print(f"Failed to fetch page {n} for {s}")
                break

            root = ET.fromstring(response.text)
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

        filename = f"courses/{s}.json"
        with open(filename, "w", encoding="utf-8") as f:
            json.dump(courses, f, indent=4)
        print(f"Saved {len(courses)} courses to {filename}")

def autosem() -> list[int]:
    y = datetime.date.today().year
    valid = []
    years = [y - 1, y, y + 1]
    post = [10, 20, 30]
    strings = [f"3{yr}{inc}" for yr in years for inc in post]
    for s in strings:
        time.sleep(0.5)
        print(f"Trying {s}")
        try:
            dryfire = requests.get(f"https://mytimetable.mcmaster.ca/api/courses/suggestions?term={s}&cams=MCMSTiMCMST_MCMSTiMHK_MCMSTiOFF_MCMSTiCON_MCMSTiSNPOL&course_add=%20&page_num=0&sco=1&sio=1&already=&_={int(datetime.datetime.now().timestamp() * 1000)}")
            root = ET.fromstring(dryfire.text)

            rs_elements = root.findall('.//rs')

            is_error = all('(' in rs.get('info', '') for rs in rs_elements)

            if not is_error and len(rs_elements) > 0:
                print(f"FOUND VALID: {s}")
                valid.append(s)

        except (ET.ParseError, requests.RequestException):
            continue

    return valid

def main():

    s = autosem()
    get_courses(s)



if __name__ == "__main__":
    main()