import io,json,pathlib,re
import chess,chess.pgn
root=pathlib.Path(__file__).resolve().parent
results=json.loads((root/'results.json').read_text())
assert len(results)==10, 'expected ten games'
assert all(r['initial_fen']==chess.STARTING_FEN for r in results), 'standard start required'
assert sum(r['white']=='cloudflare/clef' for r in results)==5, 'colors must be balanced'
assert all(r['exit_code']==0 for r in results), 'match failed'
summary=[]
for record in results:
    name=record['game_id']
    pgn=(root/f'{name}.pgn').read_text()
    game=chess.pgn.read_game(io.StringIO(pgn))
    assert game is not None and not game.errors,(name,'PGN errors',None if game is None else game.errors)
    board=chess.Board(record['initial_fen'])
    assert board.is_valid(),(name,'invalid starting board')
    assert game.board().fen(en_passant='fen')==board.fen(en_passant='fen'),(name,'PGN initial FEN')
    pgn_moves=list(game.mainline_moves())
    assert len(pgn_moves)==record['plies']==len(record['moves']),(name,'move count')
    for i,(move_record,pgn_move) in enumerate(zip(record['moves'],pgn_moves),1):
        assert board.outcome(claim_draw=False) is None,(name,i,'continued after automatic terminal outcome')
        move=chess.Move.from_uci(move_record['uci'])
        assert move in board.legal_moves,(name,i,'illegal UCI',move)
        assert move==pgn_move,(name,i,'PGN and UCI disagree')
        assert board.san(move)==move_record['san'],(name,i,'SAN disagrees')
        assert move_record['side']==('white' if board.turn else 'black'),(name,i,'side')
        board.push(move)
        assert board.is_valid(),(name,i,'invalid resulting board')
    log=(root/f'{name}.log').read_text()
    final_fens=re.findall(r'^FEN: (.+)$',log,re.M)
    assert len(final_fens)==1,(name,'missing/duplicate final FEN')
    assert board.fen(en_passant='fen')==final_fens[0],(name,'logged final FEN mismatch',board.fen(en_passant='fen'),final_fens)
    assert game.end().board().fen(en_passant='fen')==final_fens[0],(name,'PGN final FEN mismatch')
    assert game.headers['Result']==record['result'],(name,'PGN result marker/header')
    outcome=board.outcome(claim_draw=False)
    expected={'Checkmate':chess.Termination.CHECKMATE,'FivefoldRepetition':chess.Termination.FIVEFOLD_REPETITION,'Stalemate':chess.Termination.STALEMATE,'InsufficientMaterial':chess.Termination.INSUFFICIENT_MATERIAL,'SeventyFiveMoveRule':chess.Termination.SEVENTYFIVE_MOVES}
    if record['reason'] in expected:
        assert outcome is not None and outcome.termination==expected[record['reason']],(name,'termination mismatch',outcome)
        assert outcome.result()==record['result'],(name,'outcome mismatch',outcome)
    else:
        assert record['reason']=='ply limit' and outcome is None and record['result']=='*',(name,'unexpected termination',record['reason'],outcome)
    summary.append({'game':name,'plies':len(pgn_moves),'result':record['result'],'reason':record['reason'],'validation':'PASS'})
print(json.dumps({'python_chess_version':chess.__version__,'completed_games_checked':len(summary),'total_plies_checked':sum(x['plies'] for x in summary),'games':summary},indent=2))
