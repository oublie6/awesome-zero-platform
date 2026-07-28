from pathlib import Path

path = Path("server/business/doudizhu/domain/playing/pattern.go")
text = path.read_text(encoding="utf-8")
old = '''func validatePattern(value Pattern) error {
\tif value.Version != RulesVersion || !knownKind(value.Kind) || value.SequenceLength == 0 || value.CardCount == 0 || value.MainRank > carddeck.RankBigJoker {
\t\treturn fmt.Errorf("%w: %#v", ErrInvalidPatternValue, value)
\t}
\treturn nil
}
'''
new = '''func validatePattern(value Pattern) error {
\tif value.Version != RulesVersion || !knownKind(value.Kind) || value.SequenceLength == 0 || value.CardCount == 0 || value.MainRank > carddeck.RankBigJoker || !validPatternStructure(value) {
\t\treturn fmt.Errorf("%w: %#v", ErrInvalidPatternValue, value)
\t}
\treturn nil
}

func validPatternStructure(value Pattern) bool {
\tsequenceLength := int(value.SequenceLength)
\tcardCount := int(value.CardCount)
\tordinaryRank := value.MainRank <= carddeck.RankTwo
\tsequenceRank := value.MainRank <= carddeck.RankAce && int(value.MainRank)-sequenceLength+1 >= int(carddeck.RankThree)

\tswitch value.Kind {
\tcase KindSingle:
\t\treturn sequenceLength == 1 && cardCount == 1
\tcase KindPair:
\t\treturn sequenceLength == 1 && cardCount == 2 && ordinaryRank
\tcase KindTriple:
\t\treturn sequenceLength == 1 && cardCount == 3 && ordinaryRank
\tcase KindTripleWithSingle:
\t\treturn sequenceLength == 1 && cardCount == 4 && ordinaryRank
\tcase KindTripleWithPair:
\t\treturn sequenceLength == 1 && cardCount == 5 && ordinaryRank
\tcase KindStraight:
\t\treturn sequenceLength >= 5 && cardCount == sequenceLength && sequenceRank
\tcase KindConsecutivePairs:
\t\treturn sequenceLength >= 3 && cardCount == 2*sequenceLength && sequenceRank
\tcase KindAirplane:
\t\treturn sequenceLength >= 2 && cardCount == 3*sequenceLength && sequenceRank
\tcase KindAirplaneWithSingles:
\t\treturn sequenceLength >= 2 && cardCount == 4*sequenceLength && sequenceRank
\tcase KindAirplaneWithPairs:
\t\treturn sequenceLength >= 2 && cardCount == 5*sequenceLength && sequenceRank
\tcase KindFourWithTwoSingles:
\t\treturn sequenceLength == 1 && cardCount == 6 && ordinaryRank
\tcase KindFourWithTwoPairs:
\t\treturn sequenceLength == 1 && cardCount == 8 && ordinaryRank
\tcase KindBomb:
\t\treturn sequenceLength == 1 && cardCount == 4 && ordinaryRank
\tcase KindRocket:
\t\treturn sequenceLength == 1 && cardCount == 2 && value.MainRank == carddeck.RankBigJoker
\tdefault:
\t\treturn false
\t}
}
'''
if text.count(old) != 1:
    raise SystemExit(f"expected one validatePattern block, found {text.count(old)}")
path.write_text(text.replace(old, new, 1), encoding="utf-8")
