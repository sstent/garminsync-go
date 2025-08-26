import os
import json
import time
import logging
from flask import Flask, request, jsonify, send_file
import io
from garminconnect import Garmin

app = Flask(__name__)

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s %(levelname)s: %(message)s')
logger = logging.getLogger(__name__)

# Global API client
api = None
last_init_time = 0
init_retry_interval = 60  # seconds

# Environment variables
GARMIN_EMAIL = os.getenv("GARMIN_EMAIL")
GARMIN_PASSWORD = os.getenv("GARMIN_PASSWORD")

def init_api():
    """Initializes the Garmin API client (with retries)."""
    global api, last_init_time
    if api and time.time() - last_init_time < init_retry_interval:
        return api
    
    try:
        new_api = Garmin(GARMIN_EMAIL, GARMIN_PASSWORD)
        new_api.login()
        api = new_api
        last_init_time = time.time()
        logger.info("Successfully authenticated with Garmin API")
        return api
    except Exception as e:
        logger.error(f"Error initializing Garmin API: {e}")
        if not api:
            logger.critical("Critical: API initialization failed")
        return api

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

@app.route('/activities/<activity_id>/download', methods=['GET'])
def download_activity(activity_id):
    """Endpoint to download activity data with retry logic."""
    api = init_api()
    if not api:
        return jsonify({"error": "Failed to connect to Garmin API"}), 500
        
    try:
        format = request.args.get('format', 'fit')  # Default to FIT format
        file_data = None
        
        # Implement exponential backoff retry (1s, 2s, 4s)
        for attempt in range(3):
            try:
                file_data = api.download_activity(activity_id, dl_fmt=format)
                break  # Success, break out of retry loop
            except Exception as e:
                wait = 2 ** attempt
                logger.warning(f"Download attempt {attempt+1}/3 failed, retrying in {wait}s: {str(e)}")
                time.sleep(wait)
        
        if file_data is None:
            return jsonify({"error": "Activity download failed after 3 attempts"}), 500
            
        return send_file(
            io.BytesIO(file_data),
            mimetype='application/octet-stream',
            as_attachment=True,
            download_name=f'activity_{activity_id}.{format}'
        )
    except Exception as e:
        logger.error(f"Error downloading activity: {str(e)}")
        return jsonify({"error": str(e)}), 500

@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint with authentication status."""
    if api:
        return jsonify({"status": "healthy", "auth_status": "authenticated", "service": "garmin-api"}), 200
    else:
        # Attempt to reinitialize if not tried recently
        init_api()
        if api:
            return jsonify({"status": "healthy", "auth_status": "reauthenticated", "service": "garmin-api"}), 200
        return jsonify({"status": "unhealthy", "auth_status": "unauthenticated", "service": "garmin-api"}), 503

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8081)
