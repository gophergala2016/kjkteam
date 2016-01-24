var babelify = require('babelify');
var browserify = require('browserify');
var buffer = require('vinyl-buffer');
var envify = require('envify/custom');
var exorcist = require('exorcist');
var gulp = require('gulp');
var concat = require('gulp-concat');
var prefix = require('gulp-autoprefixer');
var sass = require('gulp-sass');
var sourcemaps = require('gulp-sourcemaps');
var source = require('vinyl-source-stream');
var uglify = require('gulp-uglify');

require('babel-register');

var t_envify = ['envify', {
  'global': true,
  '_': 'purge',
  NODE_ENV: 'production'
}];

// 'plugins': ['undeclared-variables-check'],
var t_babelify = ['babelify', {
  'presets': ['es2015', 'react']
}];

gulp.task('jsvendor', function() {
  return gulp.src([
    'www/static/components/jquery/dist/jquery.min.js',
    'www/static/components/underscore/underscore.js',
    'www/static/components/highlightjs/highlight.pack.js',
    'js/codediff.js/difflib.js',
    'js/codediff.js/codediff.js',
    'js/file_diff.js'
  ])
  .pipe(concat('vendor.js'))
  .pipe(gulp.dest('www/static/dist'));
});

gulp.task('js', function() {
  browserify({
    entries: ['js/App.jsx'],
    'transform': [t_babelify],
    debug: true
  })
    .bundle()
    .pipe(exorcist('www/static/dist/bundle.js.map'))
    .pipe(source('bundle.js'))
    .pipe(gulp.dest('www/static/dist'));
});

gulp.task('jsprod', function() {
  browserify({
    entries: ['js/App.jsx'],
    'transform': [t_babelify, t_envify],
    debug: true
  })
    .bundle()
    .pipe(exorcist('www/static/dist/bundle.min.js.map'))
    .pipe(source('bundle.min.js'))
    .pipe(buffer())
    .pipe(uglify())
    .pipe(gulp.dest('www/static/dist'));
});

gulp.task('css', function() {
  return gulp.src('./sass/main.scss')
    .pipe(sourcemaps.init())
    .pipe(sass().on('error', sass.logError))
    .pipe(prefix('last 2 versions'))
    .pipe(sourcemaps.write('.')) // this is relative to gulp.dest()
    .pipe(gulp.dest('./www/static/dist/'));
});

gulp.task('watch', function() {
  gulp.watch('js/**/*js*', ['js']);
  gulp.watch(['sass/*'], ['css']);
});

gulp.task('build_and_watch', ['css', 'js', 'jsvendor', 'watch']);
gulp.task('prod', ['css', 'jsvendor', 'jsprod']);
gulp.task('default', ['css', 'jsvendor', 'js']);
