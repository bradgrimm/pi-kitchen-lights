var crypto = require('crypto');
var request = require('request');
var pubsub = require('@google-cloud/pubsub');

var pubsubClient = pubsub({ projectId: "<PUBSUB_PROJECT_ID>" });
var topicName = "kitchen_lights";

exports.triggerLights = function triggerLights(req, res) {

  let username = req.body.username;
  let password = req.body.password;
  
  if (username !== "<USERNAME>") {
    res.status(403).send('Invalid username.');
    return;
  }

  var salt = "<SALT>";
  var hash = crypto.createHmac('sha512', salt);
  hash.update(password);
  var value = hash.digest('hex');
  if (value !== "<HASHED_PASSWORD>") {
    res.status(403).send('Invalid password.');
    return;
  }
  
  delete req.body.username;
  delete req.body.password;
  
  let action = req.body.action;
  if (typeof action === 'undefined') {
    res.status(400).send('Unrecognized action.');
  }

  console.log('Performing action: ' + action);
  createTopic(function(topic) {
    publishMessage(topic, req.body, function() {
      res.status(200).end();
    });
  });
};

function createTopic(callback) {

  if (!callback) {
    console.log('No callback provided.');
    return;
  }

  pubsubClient.createTopic(topicName, function(error, topic) {

    // Topic already exists
    if (error && error.code === 409) {
      console.log('Topic created');
      callback(pubsubClient.topic(topicName));
      return;
    }

    if (error) {
      console.log(error);
      return;
    }

    callback(pubsubClient.topic(topicName));
  });
}

function publishMessage(topic, message, callback) {

  topic.publish(message, function(error) {

    if (error) {
      console.log('Publish error:');
      console.log(error);
      return;
    }

    console.log('Publish successful');

    if (callback) {
      callback();
    }
  });
}
