-- the list of all known, distinct puzzles (both input and solved)
create table puzzles(
  puzzleId text primary key,	   -- puzzle's content signature
  geometry text not null,	   -- puzzle's geometry
  sideLength int not null,	   -- puzzle's side length
  valueList int array,		   -- puzzle's values
  created timestamp with time zone -- when the puzzle was entered
  );

-- the list of all known sessions
create table sessions(
  sessionId text primary key,	    -- Heroku Request ID or smaller
  created timestamp with time zone, -- when the session was created
  updated timestamp with time zone  -- when the session was last updated
  );

-- the solutions for all the known input puzzles
create table solutions(
  puzzleId text references puzzles on delete cascade on update cascade,
  solutionId text references puzzles on delete cascade on update cascade,
  choicePairs int array, -- flattened array of <index, choice> pairs
  rating int not null,	 -- star rating of the solution (1 to 5)
  primary key (puzzleId, solutionId)
  );
-- look up solutions by puzzle
create index on solutions (puzzleId);

-- each session's list of input puzzles, with session-local metadata
create table sessionPuzzles(
  sessionId text references sessions on delete cascade on update cascade,
  puzzleId text references puzzles on delete cascade on update cascade,
  puzzleName text,		       -- this session's name for the puzzle
  lastWorked timestamp with time zone, -- when the puzzle was last worked by the user
  choicePairs int array,	       -- flattened array of choices made in the session
  primary key (sessionId, puzzleId)
  );
-- look up puzzles by session
create index on sessionPuzzles (sessionId);
