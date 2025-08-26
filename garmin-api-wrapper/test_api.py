import os
import json
from garminconnect import Garmin
import logging
from datetime import datetime

def main():
    # Setup logging
    logging.basicConfig(level=logging.INFO, format='%(asctime)s %(levelname)s: %(message)s')
    logger = logging.getLogger(__name__)
    
    print("=== Starting Garmin API Tests ===")
    
    # Load credentials from environment
    email = os.getenv("GARMIN_EMAIL")
    password = os.getenv("GARMIN_PASSWORD")
    
    if not email or not password:
        logger.error("GARMIN_EMAIL or GARMIN_PASSWORD environment variables not set")
        return
    
    try:
        # 1. Test Authentication
        logger.info("Testing authentication...")
        api = Garmin(email, password)
        api.login()
        logger.info("✅ Authentication successful")
        
        # 2. Test Activity Listing
        logger.info("Testing activity listing...")
        activities = api.get_activities(0, 1)  # Get 1 most recent activity
        if not activities:
            logger.error("❌ No activities found")
        else:
            logger.info(f"✅ Found {len(activities)} activities")
            print("Sample activity:")
            print(json.dumps(activities[0], indent=2)[:1000])  # Print first 1000 chars
        
        # 3. Test Activity Download (if we got any activities)
        if activities:
            logger.info("Testing activity download...")
            activity_id = activities[0]["activityId"]
            details = api.get_activity(activity_id)
            if details:
                logger.info(f"✅ Activity {activity_id} details retrieved")
                print("Sample details:")
                print(json.dumps(details, indent=2)[:1000])  # Print first 1000 chars
            else:
                logger.error("❌ Failed to get activity details")
        
        # 4. Test Stats Retrieval
        logger.info("Testing stats retrieval...")
        stats = api.get_stats(datetime.now().strftime("%Y-%m-%d"))
        if stats:
            logger.info("✅ Stats retrieved")
            print(json.dumps(stats, indent=2)[:1000])
        else:
            logger.error("❌ Failed to get stats")
    
    except Exception as e:
        logger.error(f"❌ Test failed: {str(e)}")
        # Print detailed exception info
        import traceback
        traceback.print_exc()
    
    print("\n=== Test Complete ===")

if __name__ == "__main__":
    main()
