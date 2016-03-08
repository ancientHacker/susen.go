create table puzzles(
	sessionId text not null,
	puzzleId text not null,
	geometry text not null,
	sideLength int not null,
	valueList int array,
	primary key (sessionId, puzzleId));
create index on puzzles (puzzleId);
