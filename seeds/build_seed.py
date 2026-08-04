#!/usr/bin/env python3
"""Generate seeds/dev_stations.sql for FireReach development.

Sources
-------
Coordinates + names : OpenStreetMap, amenity=fire_station within Ghana,
                      via the Overpass API. Licensed ODbL.
Region + district   : OSM Nominatim reverse geocoding (state, county).
                      Licensed ODbL.
Contact numbers     : Ghana National Fire Service official contact-numbers
                      page, archived 2022-08-09 at
                      web.archive.org/web/20220809162701/
                      http://www.gnfs.gov.gh/contact-numbers

This script exists so the derivation is auditable and repeatable. The SQL it
produces is committed, so `make seed` never needs network access. Re-running
against live OSM may legitimately produce a different row set.

Usage: python3 seeds/build_seed.py > seeds/dev_stations.sql
"""

import json
import sys
import time
import urllib.parse
import urllib.request
import uuid

OVERPASS = "https://overpass.kumi.systems/api/interpreter"
NOMINATIM = "https://nominatim.openstreetmap.org/reverse"
UA = "FireReach-dev-seed/1.0 (https://github.com/firereach)"

OVERPASS_QUERY = """[out:json][timeout:90];
area["ISO3166-1"="GH"][admin_level=2]->.gh;
(node["amenity"="fire_station"](area.gh);
 way["amenity"="fire_station"](area.gh););
out center tags;"""

# GNFS regional commands, verbatim from the archived official page.
# Ordered [first-published, second-published]; publication order becomes
# response_rate 1.0 / 0.5. These are NOT measured response rates.
GNFS_COMMANDS = {
    "greater accra": ["0302666576", "0299346018"],
    "ashanti":       ["0322022221", "0299346044"],
    "eastern":       ["0302982062", "0299346041"],
    "central":       ["0332132902", "0299340499"],
    "western":       ["0312193521", "0299346040"],
    "volta":         ["0362026679", "0299346042"],
    # DISCREPANCY: the official page lists the Northern landline as
    # 0322022864, but 032 is the Kumasi/Ashanti prefix while Tamale is 037;
    # an independent aggregator lists 0372022864. Since we cannot verify
    # which is correct, the undisputed 0299 number is published first and
    # therefore becomes primary.
    "northern":      ["0299346046", "0322022864"],
    "brong ahafo":   ["0352027129", "0299340249"],
    "upper east":    ["0382022277"],
    "upper west":    ["0392022389"],
    "tema":          ["0303202554", "0299340083"],
}
NATIONAL = ["0302772446", "0299340383"]  # Fire Master Control
EMERGENCY = "192"

# The archived page predates the 2019 regional reorganization, so successor
# regions inherit their predecessor's command.
REGION_TO_COMMAND = {
    "greater accra region": "greater accra",
    "ashanti region":       "ashanti",
    "eastern region":       "eastern",
    "central region":       "central",
    "western region":       "western",
    "western north region": "western",
    "volta region":         "volta",
    "oti region":           "volta",
    "northern region":      "northern",
    "savannah region":      "northern",
    "north east region":    "northern",
    "bono region":          "brong ahafo",
    "bono east region":     "brong ahafo",
    "ahafo region":         "brong ahafo",
    "upper east region":    "upper east",
    "upper west region":    "upper west",
}

NAMESPACE = uuid.UUID("6ba7b811-9dad-11d1-80b4-00c04fd430c8")  # RFC 4122 URL


def station_uuid(kind, osm_id):
    """Deterministic UUID so re-seeding preserves station identity."""
    return str(uuid.uuid5(NAMESPACE, f"https://www.openstreetmap.org/{kind}/{osm_id}"))


def fetch_stations():
    body = urllib.parse.urlencode({"data": OVERPASS_QUERY}).encode()
    req = urllib.request.Request(OVERPASS, data=body, headers={"User-Agent": UA})
    with urllib.request.urlopen(req, timeout=180) as r:
        return json.load(r).get("elements", [])


def reverse(lat, lon):
    qs = urllib.parse.urlencode({"format": "jsonv2", "lat": lat, "lon": lon, "zoom": 8})
    req = urllib.request.Request(f"{NOMINATIM}?{qs}", headers={"User-Agent": UA})
    with urllib.request.urlopen(req, timeout=30) as r:
        addr = json.load(r).get("address", {})
    return addr.get("state") or "", addr.get("county") or ""


def sql_str(value):
    return "'" + value.replace("'", "''") + "'"


def main():
    elements = fetch_stations()
    named = [e for e in elements if e.get("tags", {}).get("name")]
    named.sort(key=lambda e: e["tags"]["name"])
    sys.stderr.write(f"{len(elements)} features, {len(named)} named\n")

    rows = []
    for e in named:
        lat = e.get("lat") or e.get("center", {}).get("lat")
        lon = e.get("lon") or e.get("center", {}).get("lon")
        if lat is None or lon is None:
            continue
        region, district = reverse(lat, lon)
        time.sleep(1.2)  # Nominatim usage policy: max 1 request/second
        rows.append({
            "id": station_uuid(e["type"], e["id"]),
            "name": e["tags"]["name"],
            "region": region or "Unknown Region",
            "district": district or "Unknown District",
            "lat": lat,
            "lng": lon,
            "numbers": GNFS_COMMANDS.get(
                REGION_TO_COMMAND.get(region.strip().lower(), ""), NATIONAL
            ),
        })
        sys.stderr.write(f"  {e['tags']['name']} -> {region}\n")

    out = sys.stdout
    out.write("-- FireReach development seed. GENERATED by seeds/build_seed.py.\n")
    out.write("-- Do not edit by hand; re-run the generator instead.\n")
    out.write("--\n")
    out.write("-- Coordinates + names : OpenStreetMap (ODbL), amenity=fire_station\n")
    out.write("-- Region + district   : OSM Nominatim reverse geocoding (ODbL)\n")
    out.write("-- Contact numbers     : GNFS official contact-numbers page,\n")
    out.write("--                       archived 2022-08-09\n")
    out.write("--\n")
    out.write("-- CAVEATS\n")
    out.write("--   response_rate encodes PUBLICATION ORDER, not measured\n")
    out.write("--   reliability. No GNFS response-rate data is published. The\n")
    out.write("--   field name overstates what these values mean.\n")
    out.write("--\n")
    out.write("--   Contact numbers are regional command lines, not per-station\n")
    out.write("--   direct lines, and were archived in 2022.\n")
    out.write("--\n")
    out.write("--   NOT VERIFIED FOR EMERGENCY USE. Confirm every number directly\n")
    out.write("--   with GNFS before any production deployment.\n")
    out.write("--\n")
    out.write("-- OSM data is ODbL: attribution and share-alike obligations apply\n")
    out.write("-- to derived databases. Resolve licensing before shipping.\n\n")
    out.write("BEGIN;\n\n")
    out.write("DELETE FROM station_contacts;\n")
    out.write("DELETE FROM stations;\n\n")

    for r in rows:
        out.write(
            "INSERT INTO stations (id, name, region, district, lat, lng, active) VALUES\n"
            f"  ('{r['id']}', {sql_str(r['name'])}, {sql_str(r['region'])}, "
            f"{sql_str(r['district'])}, {r['lat']}, {r['lng']}, true);\n"
        )
        contacts = [(p, 1.0 if i == 0 else 0.5) for i, p in enumerate(r["numbers"])]
        contacts.append((EMERGENCY, 0.1))
        for phone, rate in contacts:
            out.write(
                "INSERT INTO station_contacts (station_id, phone, response_rate, active) "
                f"VALUES ('{r['id']}', '{phone}', {rate}, true);\n"
            )
        out.write("\n")

    out.write("COMMIT;\n")
    sys.stderr.write(f"wrote {len(rows)} stations\n")


if __name__ == "__main__":
    main()
