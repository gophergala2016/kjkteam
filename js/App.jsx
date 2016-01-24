import React from 'react';
import ReactRouter from 'react-router';

import './util.js';
import './image.jsx';
import { makeRoot } from './components.jsx';

var Route = ReactRouter.Route;
var Routes = ReactRouter.Routes;

export var App = React.createClass({
  render: function() {
    return <ReactRouter.RouteHandler />;
  }
});

export var routes = (
  <Route handler={App}>
    <Route name="pair" path="/:index?" handler={makeRoot(pairs, initialIdx)} />
  </Route>
);

console.log("App.jsx");

ReactRouter.run(routes, ReactRouter.HistoryLocation, function(Handler) {
  React.render(<Handler/>, $('#application').get(0));
});
