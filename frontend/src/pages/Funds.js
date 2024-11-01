import React, { useState, useEffect } from 'react';
import axios from 'axios';
import Modal from '../components/Modal';
import './Funds.css';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faMoneyBill, faChartLine } from '@fortawesome/free-solid-svg-icons';

const API_BASE_URL = 'http://localhost:5000/api';

const Funds = () => {
  const [funds, setFunds] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingFund, setEditingFund] = useState(null);
  const [newFund, setNewFund] = useState({
    name: '',
    isin: '',
    currency: 'EUR',
    exchange: ''
  });

  useEffect(() => {
    fetchFunds();
  }, []);

  const fetchFunds = async () => {
    try {
      const response = await axios.get(`${API_BASE_URL}/funds`);
      setFunds(response.data);
      setError(null);
    } catch (err) {
      setError('Error fetching funds');
      console.error('Error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      if (editingFund) {
        const response = await axios.put(
          `${API_BASE_URL}/funds/${editingFund.id}`,
          editingFund
        );
        setFunds(funds.map(f => f.id === editingFund.id ? response.data : f));
      } else {
        const response = await axios.post(`${API_BASE_URL}/funds`, newFund);
        setFunds([...funds, response.data]);
      }
      setIsModalOpen(false);
      setEditingFund(null);
      setNewFund({ name: '', isin: '', currency: 'EUR', exchange: '' });
    } catch (error) {
      console.error('Error saving fund:', error);
    }
  };

  const handleDelete = async (id) => {
    try {
      const usageResponse = await axios.get(`${API_BASE_URL}/funds/${id}/check-usage`);
      if (usageResponse.data.in_use) {
        const portfolioInfo = usageResponse.data.portfolios
          .map(p => `${p.name} (${p.transaction_count} transactions)`)
          .join('\n');
        alert(`Cannot delete fund because it has transactions in the following portfolios:\n\n${portfolioInfo}`);
        return;
      }

      if (window.confirm('Are you sure you want to delete this fund?')) {
        await axios.delete(`${API_BASE_URL}/funds/${id}`);
        setFunds(funds.filter(f => f.id !== id));
      }
    } catch (error) {
      console.error('Error deleting fund:', error);
      alert('Error deleting fund: ' + (error.response?.data?.error || 'Unknown error'));
    }
  };

  const getDividendTypeDisplay = (type) => {
    switch(type) {
      case 'cash':
        return <><FontAwesomeIcon icon={faMoneyBill} /> Cash Dividend</>;
      case 'stock':
        return <><FontAwesomeIcon icon={faChartLine} /> Stock Dividend</>;
      case 'none':
      default:
        return 'No Dividend';
    }
  };

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;

  return (
    <div className="funds-page">
      <div className="page-header">
        <h1>Funds</h1>
        <button onClick={() => setIsModalOpen(true)}>Add Fund</button>
      </div>

      <div className="funds-grid">
        {funds.map(fund => (
          <div key={fund.id} className="fund-card">
            <h2>{fund.name}</h2>
            <div className="fund-details">
              <p><strong>ISIN:</strong> {fund.isin}</p>
              <p><strong>Currency:</strong> {fund.currency}</p>
              <p><strong>Exchange:</strong> {fund.exchange}</p>
              <p>
                <strong>Dividend Type: </strong>
                {getDividendTypeDisplay(fund.dividend_type)}
              </p>
            </div>
            <div className="fund-actions">
              <button onClick={() => {
                setEditingFund(fund);
                setIsModalOpen(true);
              }}>Edit</button>
              <button onClick={() => handleDelete(fund.id)}>Delete</button>
            </div>
          </div>
        ))}
      </div>

      <Modal 
        isOpen={isModalOpen}
        onClose={() => {
          setIsModalOpen(false);
          setEditingFund(null);
        }}
        title={editingFund ? "Edit Fund" : "Add Fund"}
      >
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Name:</label>
            <input
              type="text"
              value={editingFund ? editingFund.name : newFund.name}
              onChange={(e) => {
                if (editingFund) {
                  setEditingFund({...editingFund, name: e.target.value});
                } else {
                  setNewFund({...newFund, name: e.target.value});
                }
              }}
              required
            />
          </div>
          <div className="form-group">
            <label>ISIN:</label>
            <input
              type="text"
              value={editingFund ? editingFund.isin : newFund.isin}
              onChange={(e) => {
                if (editingFund) {
                  setEditingFund({...editingFund, isin: e.target.value});
                } else {
                  setNewFund({...newFund, isin: e.target.value});
                }
              }}
              required
            />
          </div>
          <div className="form-group">
            <label>Currency:</label>
            <input
              type="text"
              value={editingFund ? editingFund.currency : newFund.currency}
              onChange={(e) => {
                if (editingFund) {
                  setEditingFund({...editingFund, currency: e.target.value});
                } else {
                  setNewFund({...newFund, currency: e.target.value});
                }
              }}
              required
            />
          </div>
          <div className="form-group">
            <label>Exchange:</label>
            <input
              type="text"
              value={editingFund ? editingFund.exchange : newFund.exchange}
              onChange={(e) => {
                if (editingFund) {
                  setEditingFund({...editingFund, exchange: e.target.value});
                } else {
                  setNewFund({...newFund, exchange: e.target.value});
                }
              }}
              required
            />
          </div>
          {editingFund && (
            <div className="form-group">
              <label>Dividend Type:</label>
              <select
                value={editingFund.dividend_type || 'none'}
                onChange={(e) => {
                  setEditingFund({
                    ...editingFund,
                    dividend_type: e.target.value
                  });
                }}
              >
                <option value="none">No Dividend</option>
                <option value="cash">
                  <FontAwesomeIcon icon={faMoneyBill} /> Cash Dividend
                </option>
                <option value="stock">
                  <FontAwesomeIcon icon={faChartLine} /> Stock Dividend
                </option>
              </select>
            </div>
          )}
          <div className="modal-actions">
            <button type="submit">{editingFund ? "Update" : "Create"}</button>
            <button type="button" onClick={() => {
              setIsModalOpen(false);
              setEditingFund(null);
            }}>Cancel</button>
          </div>
        </form>
      </Modal>
    </div>
  );
};

export default Funds;
