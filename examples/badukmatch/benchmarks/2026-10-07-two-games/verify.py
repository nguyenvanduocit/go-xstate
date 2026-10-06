import json, pathlib, re
from sgfmill import boards, sgf
root = pathlib.Path(__file__).resolve().parent
cols = 'ABCDEFGHJKLMNOPQRST'
def cells(board):
    return ''.join({None: '.', 'b': 'X', 'w': 'O'}[board.get(r,c)] for r in range(board.side-1,-1,-1) for c in range(board.side))
results = []
for game_id in (1, 2):
    stem = f'{game_id:02d}'
    execution = json.loads((root / (stem+'-execution.json')).read_text())
    game = sgf.Sgf_game.from_bytes((root / (stem+'.sgf')).read_bytes())
    log = (root / (stem+'.log')).read_text()
    entries = re.findall(r'^\s*(\d+) (black|white)\s+(\S+)\s+(\S+)\s+confidence=([\d.]+) cost=\$([\d.]+) latency=(\d+)ms$', log, re.M)
    nodes = list(game.get_main_sequence())[1:]
    assert len(nodes) == len(entries)
    board = boards.Board(game.get_size())
    history = {cells(board)}
    consecutive_passes = 0
    stats = {color: {'model': execution['black' if color == 'b' else 'white'], 'moves': 0, 'passes': 0, 'captured_opponent_stones': 0, 'self_captured_stones': 0, 'sum_rounded_move_cost_usd': 0, 'latency_ms': 0} for color in ('b', 'w')}
    for i, (node, entry) in enumerate(zip(nodes, entries)):
        color, point = node.get_move()
        assert color == ('b' if i%2 == 0 else 'w')
        assert consecutive_passes < 2
        number, side, model, move, confidence, cost, latency = entry
        assert int(number) == i+1 and side == ('black' if color == 'b' else 'white')
        assert model == stats[color]['model']
        assert move == ('pass' if point is None else cols[point[1]]+str(point[0]+1))
        own, opponent = ('X','O') if color == 'b' else ('O','X')
        before = cells(board)
        if point is None:
            consecutive_passes += 1
            stats[color]['passes'] += 1
        else:
            r, c = point
            assert board.get(r,c) is None
            board.play(r,c,color)
            after = cells(board)
            assert after not in history, f'superko game {game_id} move {i+1}'
            history.add(after)
            stats[color]['captured_opponent_stones'] += before.count(opponent)-after.count(opponent)
            stats[color]['self_captured_stones'] += before.count(own)+1-after.count(own)
            consecutive_passes = 0
        stats[color]['moves'] += 1
        stats[color]['sum_rounded_move_cost_usd'] += float(cost)
        stats[color]['latency_ms'] += int(latency)
    final_cells = cells(board)
    printed_rows = re.findall(r'^\s*[1-9]\s+((?:[XO.] ){9})$', log, re.M)
    assert ''.join(row.replace(' ','') for row in printed_rows) == final_cells
    result_match = re.search(r'Result: (\S+) \(([^)]+)\), (\d+) moves, reported cost \$([\d.]+)', log)
    assert result_match
    result, reason, count, total_cost = result_match.groups()
    assert int(count) == len(nodes)
    difference = board.area_score() - game.get_komi()
    if consecutive_passes == 2:
        expected = ('B+' if difference > 0 else 'W+') + f'{abs(difference):g}' if difference != 0 else '0'
        assert result == expected == game.get_root().get('RE')
    else:
        assert result == '*' and not game.get_root().has_property('RE')
    for value in stats.values():
        value['sum_rounded_move_cost_usd'] = round(value['sum_rounded_move_cost_usd'], 6)
        value['mean_latency_ms'] = round(value['latency_ms']/value['moves'],1) if value['moves'] else None
    results.append({**execution, 'moves': len(nodes), 'result': result, 'reason': reason, 'winner': stats['b' if difference > 0 else 'w']['model'] if consecutive_passes == 2 and difference else None, 'reported_cost_usd': float(total_cost), 'black_minus_white_after_komi': difference, 'completed': consecutive_passes == 2, 'stats': stats, 'final_board': final_cells, 'sgf': stem+'.sgf', 'log': stem+'.log'})
(root / 'results.json').write_text(json.dumps(results, indent=2)+'\n')
validation = {'validator': 'sgfmill 1.1.1 with independent positional-history check', 'games': len(results), 'moves_verified': sum(x['moves'] for x in results), 'checks': ['alternating colors', 'log versus SGF moves and player models', 'empty target points', 'captures and self-captures', 'positional superko', 'two-pass termination', 'final board versus log', 'area score plus komi versus SGF and log result'], 'passed': True}
(root / 'validation.json').write_text(json.dumps(validation, indent=2)+'\n')
print(json.dumps(results, indent=2))
print(json.dumps(validation))
