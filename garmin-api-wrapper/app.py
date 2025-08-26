import os
import json
from flask import Flask, request, jsonify
from garminconnect import Garmin
import logging

app = Flask(__name__)

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s %(levelname)s: %(message)s')
logger = logging.getLogger(__name__)

# Environment variables
GARMIN_EMAIL = os.getenv("GARMIN_EMAIL")
GARMIN_PASSWORD = os.getenv("GARMIN_PASSWORD")

def init_api():
    """Initializes the Garmin API client."""
    try:
        api = Garmin(GARMIN_EMAIL, GARMIN_PASSWORD)
        api.login()
        logger.info("Successfully authenticated with Garmin API")
        return api
    except Exception as e:
        logger.error(f"Error initializing Garmin API: {e}")
        return None

@app.route('/stats', methods=['GET'])
def get_stats():
    """Endpoint to get user stats."""
    stats_date = request.args.get('date')
    if not stats_date:
        return jsonify({"error": "A 'date' query parameter is required in YYYY-MM-DD format."}), 400

    api = init_api()
    if not api:
        return jsonify({"error": "Failed to connect to Garmin API"}), 500

    try:
        logger.info(f"Fetching stats for date: {stats_date}")
        user_stats = api.get_stats(stats_date)
        return jsonify(user_stats)
    except Exception as e:
        logger.error(f"Error fetching stats: {str(e)}")
        return jsonify({"error": str(e)}), 500

@app.route('/activities', methods=['GET'])
def get_activities():
    """Endpoint to get activities list."""
    start = request.args.get('start', default=0, type=int)
    limit = request.args.get('limit', default=10, type=int)
    
    api = init_api()
    if not api:
        return jsonify({"error": "Failed to connect to Garmin API"}), 500

    try:
        logger.info(f"Fetching activities from {start} with limit {limit}")
        activities = api.get_activities(start, limit)
        return jsonify(activities)
    except Exception as e:
        logger.error(f"Error fetching activities: {str(e)}")
        return jsonify({"error": str(e)}), 500

@app.route('/activities/<activity_id>', methods=['GET'])
def get_activity_details(activity_id):
    """Endpoint to get activity details."""
    api = init_api()
    if not api:
        return jsonify({"error": "Failed to connect to Garmin API"}), 500

    try:
        logger.info(f"Fetching activity details for {activity_id}")
        activity = api.get_activity(activity_id)
        return jsonify(activity)
    except Exception as e:
        logger.error(f"Error fetching activity details: {str(e)}")
        return jsonify({"error": str(e)}), 500

@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint."""
    return jsonify({"status": "healthy", "service": "garmin-api"})

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8081)
