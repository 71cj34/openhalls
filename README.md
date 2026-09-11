# OpenHalls 

OpenHalls is a project that provides a way to automatically scrape the MyTimetable API for course time data, sort it into JSON files, and provide a fast, clean, and easy-to-use CLI in Go to view the data. Includes CSV exports, favorites in-app, SQLite database building for fast and exportable schedules, convenient interface, and four different query types (including custom SQL query engine!).

This tool is very helpful if you want to sit in on a course's lectures, find out what times a room is occupied, learn more about a course you want to take, plan sections, find a calm area for a club meeting, get analytics on room/time use, and more!

# Usage

## Scraping data (Python)

Requires Python 3.10+.

Run this: 

```bash
pip install -r requirements.txt
```
(if you get an error that pyfiglet is not compatible, then ignore it. It's not technically necessary for this project.)

Then run these commands:

```bash
# 1. Get valid semesters (--auto finds current terms, alternatively, specify one semester)
python api.py --auto
python api.py --sem 3202630 # pattern: 3 + year + semester number + 0 where semester number: winter = 1, spring/summer=2, fall=3

# 2-4. Class times + schedules (!! needs your browser cookie, see below)
python parse.py --semester 3202630 --cookie "..."

# The --no-fetch option assumes you already downloaded the data but the next step failed: this lets you retry from there
python parse.py --semester 3202630 --no-fetch

# Test n courses using --limit, specify a cookie file using --cookie-file, specify a course file with --course-file, force a reprocess with --force
python parse.py --semester 3202630 --cookie-file cookie.txt --course-file sample_data.json --limit 5 --force
```

These commands will take a hot second to run, especially `parse.py` (there are ~2000 courses to fetch per semester). If you're really impatient, you can use the `--delay` parameter (default 0.25) to lower the delay between fetches, at the risk of catching a rate limit.
 
### Getting a cookie

Getting class data only works if you are logged in to MyTimetable. Since this script isn't a browser, you need to supply your browser cookie (basically your browser's authentication key to prove that you're you.)

1. Log in at https://mytimetable.mcmaster.ca in your browser.
2. Open DevTools (F12) > Network tab, search for any course and select it to add it to your timetable.
3. Click the request with `class-data` in its `File` > Request Headers > right click, Copy Value.
4. Use one of: `--cookie "..."`, `--cookie-file cookie.txt`,
   `MYTIMETABLE_COOKIE="..."` env var, or a `.env` file with `MYTIMETABLE_COOKIE=...`.

I don't know how long it takes, but the cookie will probably expire. If the script stops working, try readding your cookie.

## CLI

This repo contains a command-line interface to view the data using SQLite for super fast browsing. To use it, download a .exe from the releases tab on Github, or compile it yourself with the `build.bat` file in \src\hallview if you have Go installed.

Once you have the .exe, place it somewhere in the folder structure between the root folder (the folder containing the .xml, .json, python files, etc) and the Go source files. Run it. The program will give you instructions, troubleshooting, etc from there.

### Why is all this so complicated?

Because this stuff is all only accesible to McMaster students, and it's kind of a security risk for me to just give out all the information about every course. Also, it's easier to maintain once I inevitably leave this school and can no longer test/update the script since I don't have credentials anymore.

Also I need a project for my portfolio in a backend language. And CLIs are baller. That too.


# Contributing

Hi! Thanks for being interested in contributing to this repo.

Submit a PR with a summary of your changes and we'll review it and get back to you ASAP.
