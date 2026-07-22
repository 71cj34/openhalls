import datetime
import re
import time
from typing import Optional
import xml.etree.ElementTree as ET
import requests
import json

def get_courses(sems: list[int]) -> Optional[list]:

    for attempt in range(3):
        try:
            url = "https://mytimetable.mcmaster.ca/api/courses/suggestions"

            n = 0
            params = {
                "term": sems[0],
                "cams": "MCMSTiMCMST_MCMSTiMHK_MCMSTiOFF_MCMSTiCON_MCMSTiSNPOL",
                "course_add": " ",
                "page_num": n,
                "sco": 1,
                "sio": 1,
                "already": "",
                "_": int(datetime.datetime.now().timestamp() * 1000),
            }

            headers = {
                "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
                "Accept": "text/xml",
            }

            cval = "20"
            courses = []

            # Notice cval is a string '20' matching root.text.strip()
            for s in sems:
                params["term"] = s
                while cval == "20":
                    # Update page number parameter on every iteration
                    params["page_num"] = n
                    # Refresh timestamp to prevent potential cached responses
                    params["_"] = int(datetime.datetime.now().timestamp() * 1000)

                    time.sleep(1)
                    response = requests.get(url, params=params, headers=headers)
                    n += 1

                    print(f"Fetching page {params['page_num']}...")

                    if response.status_code == 200:
                        if not response.text:
                            raise requests.RequestException("No response received")

                        root = ET.fromstring(response.text)
                        cval = root.text.strip() if root.text else ""

                        for item in root.findall(".//rs"):
                            course_code = item.text.strip() if item.text else ""

                            # Skip placeholder pagination items
                            if course_code.startswith("_") or not course_code:
                                continue

                            raw_info = item.get("info", "")
                            clean_info = re.sub(r"<[^>]+>", " ", raw_info).strip()

                            if clean_info.endswith("-"):
                                clean_info = clean_info[:-1].strip()

                            courses.append(
                                {"code": course_code, "title": clean_info}
                            )
                    else:
                        print(f"Failed with code {response.status_code}")
                        break

            return courses
        except Exception as e:
            print(f"Attempt {attempt + 1} failed due to: {e}")

    return None

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
    result = get_courses(s)
    if result:
        try:
            with open("output.txt", "w", encoding="utf-8") as f:
                json.dump(result, f, indent=4)
            print("Successfully wrote results to output.txt")
        except IOError as e:
            print(f"Error writing to file: {e}")

    else:
        print("API call failed or stopped")

if __name__ == "__main__":
    main()