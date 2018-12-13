import os
import time
import json
import requests
import logging

from google.cloud import pubsub_v1

SUBSCRIPTION_NAME = 'fpp'
LOG_PATH = '/home/fpp/christmas-lights/christmas.log'


def do_action(action):
    logging.info('Action: {}'.format(action))
    if action == 'on':
        response = requests.get('http://fpp/fppxml.php?command=startPlaylist&playList=Always_On&repeat=checked&playEntry=0&section=')
    elif action == 'start-show':
        response = requests.get('http://fpp/fppxml.php?command=startPlaylist&playList=Main%20Playlist&repeat=checked&playEntry=0&section=')
    elif action == 'off':
        response = requests.get('http://fpp/fppxml.php?command=stopGracefully')
    else:
        logging.warn('Unrecognized action: {}'.format(action))
        return
    logging.info('response: {}'.format(response))


def callback(message):
    print('Received message: {}'.format(message))
    message.ack()
    try:
        payload = json.loads(message.data)
        action = payload['action']
        do_action(action)
        print('Received action: {}'.format(action))
    except Exception as err:
        logging.error('Message failed: {}'.format(err))
        print('Message failed: {}'.format(err))


def main():
    project_id = os.environ['GOOGLE_PROJECT_ID']
    subscriber = pubsub_v1.SubscriberClient()
    subscription_path = subscriber.subscription_path(project_id, SUBSCRIPTION_NAME)
    subscriber.subscribe(subscription_path, callback=callback)

    print('Listening to messages on: {}'.format(subscription_path))
    logging.info('Listening to messages on: {}'.format(subscription_path))
    while True:
        time.sleep(60)


if __name__ == '__main__':
    if os.path.exists(LOG_PATH):
        os.remove(LOG_PATH)
    logging.basicConfig(filename=LOG_PATH, level=logging.INFO)
    main()
