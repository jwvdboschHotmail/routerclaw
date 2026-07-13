import urllib.request
import urllib.parse
import json
import sys
from datetime import datetime, timedelta, timezone

client_id = "YOUR_CLIENT_ID.apps.googleusercontent.com"
client_secret = "YOUR_CLIENT_SECRET"
refresh_token = "YOUR_REFRESH_TOKEN"

data = urllib.parse.urlencode({"client_id": client_id, "client_secret": client_secret, "refresh_token": refresh_token, "grant_type": "refresh_token"}).encode()
req = urllib.request.Request("https://oauth2.googleapis.com/token", data=data)
r = urllib.request.urlopen(req)
access_token = json.loads(r.read())["access_token"]

now = datetime.now(timezone.utc) - timedelta(hours=2)
endOfDay = now + timedelta(days=5)

url = f"https://www.googleapis.com/calendar/v3/calendars/primary/events?timeMin={urllib.parse.quote(now.isoformat())}&timeMax={urllib.parse.quote(endOfDay.isoformat())}&singleEvents=true&orderBy=startTime"
req2 = urllib.request.Request(url, headers={"Authorization": f"Bearer {access_token}"})
r2 = urllib.request.urlopen(req2)
events = json.loads(r2.read()).get("items", [])
for e in events:
    print(f"[{e.get('start', {}).get('dateTime')}] {e.get('summary')}")
